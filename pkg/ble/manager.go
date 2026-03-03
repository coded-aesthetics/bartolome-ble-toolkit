// Package ble provides a simplified, reliable BLE device manager
// This version focuses on stability and simplicity over complex features
package ble

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// SimpleManager handles BLE device connections with automatic reconnect support
type SimpleManager struct {
	adapter           *bluetooth.Adapter
	connected         map[string]*SimpleDevice
	addressToName     map[string]string
	pendingConfigs    map[string]DeviceConfig
	disconnectHandler func(deviceName string, address string, err error)
	mu                sync.RWMutex
	enabled           bool
	closing           bool
}

// SimpleDevice represents a connected BLE device
type SimpleDevice struct {
	Name           string
	Address        bluetooth.Address
	Device         *bluetooth.Device
	Channel        <-chan []byte
	rawChannel     chan []byte
	disconnectFunc func()
	closeOnce      sync.Once
}

func (d *SimpleDevice) closeChannel() {
	d.closeOnce.Do(func() {
		close(d.rawChannel)
	})
}

// NewSimpleManager creates a new simplified BLE manager
func NewSimpleManager() *SimpleManager {
	return &SimpleManager{
		adapter:        bluetooth.DefaultAdapter,
		connected:      make(map[string]*SimpleDevice),
		addressToName:  make(map[string]string),
		pendingConfigs: make(map[string]DeviceConfig),
	}
}

// SetDisconnectHandler sets the callback for device disconnections
func (m *SimpleManager) SetDisconnectHandler(handler func(deviceName string, address string, err error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectHandler = handler
}

// enable initializes the BLE adapter and registers the connect/disconnect handler.
// Must be called before any Connect calls.
func (m *SimpleManager) enable() error {
	m.mu.Lock()
	if m.enabled {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	fmt.Println("🔌 Enabling BLE adapter...")
	if err := m.adapter.Enable(); err != nil {
		return fmt.Errorf("failed to enable adapter: %v", err)
	}

	// Give macOS time to initialize
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled")

	// Must be set before adapter.Connect() calls per tinygo/bluetooth docs
	m.adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		if !connected {
			m.handleDisconnect(device)
		}
	})

	m.mu.Lock()
	m.enabled = true
	m.mu.Unlock()

	return nil
}

// handleDisconnect is called by the adapter when a peripheral disconnects.
func (m *SimpleManager) handleDisconnect(device bluetooth.Device) {
	addrStr := device.Address.String()

	m.mu.Lock()
	name, ok := m.addressToName[addrStr]
	if !ok {
		m.mu.Unlock()
		return
	}

	simpleDevice := m.connected[name]
	config, hasConfig := m.pendingConfigs[name]
	isClosing := m.closing

	delete(m.connected, name)
	delete(m.addressToName, addrStr)
	m.mu.Unlock()

	// Close the channel to unblock the handleNotifications goroutine
	if simpleDevice != nil {
		simpleDevice.closeChannel()
	}

	fmt.Printf("⚠️  Device %s [%s] disconnected\n", name, addrStr)

	if m.disconnectHandler != nil {
		m.disconnectHandler(name, addrStr, nil)
	}

	if hasConfig && !isClosing {
		fmt.Printf("🔄 Will attempt to reconnect to %s...\n", name)
		go m.reconnectLoop(config)
	}
}

// reconnectLoop continuously attempts to reconnect until successful or closing.
func (m *SimpleManager) reconnectLoop(config DeviceConfig) {
	for {
		m.mu.RLock()
		isClosing := m.closing
		m.mu.RUnlock()

		if isClosing {
			return
		}

		time.Sleep(3 * time.Second)

		fmt.Printf("🔄 Reconnecting to %s...\n", config.Name)
		if err := m.connectDevice(config); err != nil {
			fmt.Printf("❌ Reconnect to %s failed: %v\n", config.Name, err)
			continue
		}

		fmt.Printf("✅ Reconnected to %s!\n", config.Name)
		return
	}
}

// ConnectToDevice connects to a single device by name and service UUID
func (m *SimpleManager) ConnectToDevice(deviceName string, serviceUUID, characteristicUUID bluetooth.UUID, notificationHandler func(string, []byte) error) error {
	if err := m.enable(); err != nil {
		return err
	}

	config := DeviceConfig{
		Name:                deviceName,
		ServiceUUID:         serviceUUID,
		CharacteristicUUID:  characteristicUUID,
		NotificationHandler: notificationHandler,
	}

	m.mu.Lock()
	m.pendingConfigs[deviceName] = config
	m.mu.Unlock()

	return m.connectDevice(config)
}

// connectDevice performs the scan + connect + notification setup for one device.
func (m *SimpleManager) connectDevice(config DeviceConfig) error {
	result, err := m.scanForDevice(config.Name)
	if err != nil {
		return err
	}

	fmt.Printf("🔗 Connecting to %s [%s]...\n", config.Name, result.Address.String())
	device, rawChannel, err := m.connectAndSetup(result, config.ServiceUUID, config.CharacteristicUUID)
	if err != nil {
		return err
	}

	simpleDevice := &SimpleDevice{
		Name:       config.Name,
		Address:    result.Address,
		Device:     device,
		rawChannel: rawChannel,
		Channel:    rawChannel,
		disconnectFunc: func() {
			device.Disconnect()
		},
	}

	m.mu.Lock()
	m.connected[config.Name] = simpleDevice
	m.addressToName[result.Address.String()] = config.Name
	m.mu.Unlock()

	go m.handleNotifications(simpleDevice, config.NotificationHandler)

	fmt.Printf("🎉 %s connected and ready!\n", config.Name)
	return nil
}

// scanForDevice scans for a specific device by name
func (m *SimpleManager) scanForDevice(deviceName string) (bluetooth.ScanResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	found := make(chan bluetooth.ScanResult, 1)
	scanErr := make(chan error, 1)

	go func() {
		err := m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.LocalName() == deviceName {
				fmt.Printf("📱 Found %s [%s] RSSI: %d\n", deviceName, result.Address.String(), result.RSSI)
				adapter.StopScan()
				found <- result
			}
		})
		if err != nil {
			scanErr <- err
		}
	}()

	select {
	case result := <-found:
		return result, nil
	case err := <-scanErr:
		return bluetooth.ScanResult{}, fmt.Errorf("scan failed: %v", err)
	case <-ctx.Done():
		m.adapter.StopScan()
		return bluetooth.ScanResult{}, fmt.Errorf("device %s not found within 30 seconds", deviceName)
	}
}

// connectAndSetup establishes connection and sets up notifications
func (m *SimpleManager) connectAndSetup(result bluetooth.ScanResult, serviceUUID, characteristicUUID bluetooth.UUID) (*bluetooth.Device, chan []byte, error) {
	// Brief delay after stopping scan (important for macOS)
	time.Sleep(500 * time.Millisecond)

	// Connect to device
	device, err := m.adapter.Connect(result.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connection failed: %v", err)
	}
	fmt.Println("✅ Device connected")

	// Discover services
	fmt.Println("🔍 Discovering services...")
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		device.Disconnect()
		return nil, nil, fmt.Errorf("service discovery failed: %v", err)
	}

	if len(services) == 0 {
		device.Disconnect()
		return nil, nil, fmt.Errorf("required service not found")
	}

	service := services[0]
	fmt.Printf("✅ Found service: %s\n", service.UUID().String())

	// Discover characteristics
	fmt.Println("🔍 Discovering characteristics...")
	characteristics, err := service.DiscoverCharacteristics([]bluetooth.UUID{characteristicUUID})
	if err != nil {
		device.Disconnect()
		return nil, nil, fmt.Errorf("characteristic discovery failed: %v", err)
	}

	if len(characteristics) == 0 {
		device.Disconnect()
		return nil, nil, fmt.Errorf("required characteristic not found")
	}

	characteristic := characteristics[0]
	fmt.Printf("✅ Found characteristic: %s\n", characteristic.UUID().String())

	// Setup notifications
	fmt.Println("🔔 Setting up notifications...")
	rawChannel := make(chan []byte, 10)

	err = characteristic.EnableNotifications(func(data []byte) {
		select {
		case rawChannel <- data:
		default:
			// Channel full, drop data to prevent blocking
			fmt.Println("⚠️  Notification dropped - channel full")
		}
	})

	if err != nil {
		device.Disconnect()
		return nil, nil, fmt.Errorf("failed to enable notifications: %v", err)
	}

	fmt.Println("✅ Notifications enabled")
	return &device, rawChannel, nil
}

// handleNotifications processes incoming notifications until the channel is closed.
func (m *SimpleManager) handleNotifications(device *SimpleDevice, handler func(string, []byte) error) {
	for data := range device.Channel {
		if handler != nil {
			if err := handler(device.Name, data); err != nil {
				fmt.Printf("⚠️  Notification handler error for %s: %v\n", device.Name, err)
			}
		}
	}
}

// IsConnected checks if a device is connected
func (m *SimpleManager) IsConnected(deviceName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connected[deviceName]
	return exists
}

// GetConnectedDevices returns a map of connected devices
func (m *SimpleManager) GetConnectedDevices() map[string]*SimpleDevice {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*SimpleDevice)
	for name, device := range m.connected {
		result[name] = device
	}
	return result
}

// Disconnect disconnects a specific device and cancels any pending reconnect.
func (m *SimpleManager) Disconnect(deviceName string) error {
	m.mu.Lock()
	device, exists := m.connected[deviceName]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("device %s not connected", deviceName)
	}

	// Remove from maps so reconnect loop won't fire
	delete(m.pendingConfigs, deviceName)
	delete(m.connected, deviceName)
	delete(m.addressToName, device.Address.String())
	m.mu.Unlock()

	if device.disconnectFunc != nil {
		device.disconnectFunc()
	}
	device.closeChannel()

	fmt.Printf("✅ Disconnected from %s\n", deviceName)
	return nil
}

// Close disconnects all devices and prevents further reconnects.
func (m *SimpleManager) Close() error {
	m.mu.Lock()
	m.closing = true

	devices := make([]*SimpleDevice, 0, len(m.connected))
	for _, device := range m.connected {
		devices = append(devices, device)
	}
	m.connected = make(map[string]*SimpleDevice)
	m.addressToName = make(map[string]string)
	m.mu.Unlock()

	for _, device := range devices {
		if device.disconnectFunc != nil {
			device.disconnectFunc()
		}
		device.closeChannel()
		fmt.Printf("✅ Disconnected from %s\n", device.Name)
	}

	return nil
}

// DeviceConfig holds configuration for a BLE device
type DeviceConfig struct {
	Name                string
	ServiceUUID         bluetooth.UUID
	CharacteristicUUID  bluetooth.UUID
	NotificationHandler func(deviceName string, data []byte) error
}

// Manager provides backward compatibility with the old interface
type Manager struct {
	simpleManager *SimpleManager
}

// NewManager creates a new BLE manager (backward compatibility)
func NewManager() *Manager {
	return &Manager{
		simpleManager: NewSimpleManager(),
	}
}

// SetDisconnectHandler sets the disconnect handler (backward compatibility)
func (m *Manager) SetDisconnectHandler(handler func(deviceName string, address string, err error)) {
	m.simpleManager.SetDisconnectHandler(handler)
}

// ConnectDevices connects to multiple devices (backward compatibility)
func (m *Manager) ConnectDevices(configs []DeviceConfig) error {
	if len(configs) == 0 {
		return fmt.Errorf("no devices to connect")
	}

	for _, config := range configs {
		err := m.simpleManager.ConnectToDevice(
			config.Name,
			config.ServiceUUID,
			config.CharacteristicUUID,
			config.NotificationHandler,
		)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %v", config.Name, err)
		}
	}

	return nil
}

// IsConnected checks if a device is connected (backward compatibility)
func (m *Manager) IsConnected(deviceName string) bool {
	return m.simpleManager.IsConnected(deviceName)
}

// GetConnectedDevices returns connected devices (backward compatibility)
func (m *Manager) GetConnectedDevices() map[string]*SimpleDevice {
	return m.simpleManager.GetConnectedDevices()
}

// Close disconnects all devices (backward compatibility)
func (m *Manager) Close() error {
	return m.simpleManager.Close()
}
