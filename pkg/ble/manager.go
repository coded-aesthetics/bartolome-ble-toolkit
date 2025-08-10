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

// SimpleManager handles BLE device connections with a focus on reliability
type SimpleManager struct {
	adapter           *bluetooth.Adapter
	connected         map[string]*SimpleDevice
	disconnectHandler func(deviceName string, address string, err error)
	mu                sync.RWMutex
}

// SimpleDevice represents a connected BLE device
type SimpleDevice struct {
	Name           string
	Address        bluetooth.Address
	Device         *bluetooth.Device
	Channel        <-chan []byte
	disconnectFunc func()
}

// NewSimpleManager creates a new simplified BLE manager
func NewSimpleManager() *SimpleManager {
	return &SimpleManager{
		adapter:   bluetooth.DefaultAdapter,
		connected: make(map[string]*SimpleDevice),
	}
}

// SetDisconnectHandler sets the callback for device disconnections
func (m *SimpleManager) SetDisconnectHandler(handler func(deviceName string, address string, err error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectHandler = handler
}

// ConnectToDevice connects to a single device by name and service UUID
func (m *SimpleManager) ConnectToDevice(deviceName string, serviceUUID, characteristicUUID bluetooth.UUID, notificationHandler func(string, []byte) error) error {
	// Enable adapter
	fmt.Println("🔌 Enabling BLE adapter...")
	if err := m.adapter.Enable(); err != nil {
		return fmt.Errorf("failed to enable adapter: %v", err)
	}

	// Give macOS time to initialize
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled")

	// Scan for device
	fmt.Printf("🔍 Scanning for %s...\n", deviceName)
	result, err := m.scanForDevice(deviceName)
	if err != nil {
		return err
	}

	// Connect to device
	fmt.Printf("🔗 Connecting to %s [%s]...\n", deviceName, result.Address.String())
	device, channel, err := m.connectAndSetup(result, serviceUUID, characteristicUUID)
	if err != nil {
		return err
	}

	// Create simple device wrapper
	simpleDevice := &SimpleDevice{
		Name:    deviceName,
		Address: result.Address,
		Device:  device,
		Channel: (<-chan []byte)(channel),
		disconnectFunc: func() {
			device.Disconnect()
		},
	}

	// Store connected device
	m.mu.Lock()
	m.connected[deviceName] = simpleDevice
	m.mu.Unlock()

	// Start notification handler
	go m.handleNotifications(simpleDevice, notificationHandler)

	fmt.Printf("🎉 %s connected and ready!\n", deviceName)
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
func (m *SimpleManager) connectAndSetup(result bluetooth.ScanResult, serviceUUID, characteristicUUID bluetooth.UUID) (*bluetooth.Device, <-chan []byte, error) {
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
	channel := make(chan []byte, 10)

	err = characteristic.EnableNotifications(func(data []byte) {
		select {
		case channel <- data:
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
	return &device, channel, nil
}

// handleNotifications processes incoming notifications
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

// Disconnect disconnects a specific device
func (m *SimpleManager) Disconnect(deviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	device, exists := m.connected[deviceName]
	if !exists {
		return fmt.Errorf("device %s not connected", deviceName)
	}

	if device.disconnectFunc != nil {
		device.disconnectFunc()
	}

	delete(m.connected, deviceName)
	fmt.Printf("✅ Disconnected from %s\n", deviceName)
	return nil
}

// Close disconnects all devices and cleans up
func (m *SimpleManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, device := range m.connected {
		if device.disconnectFunc != nil {
			device.disconnectFunc()
		}
		fmt.Printf("✅ Disconnected from %s\n", name)
	}

	m.connected = make(map[string]*SimpleDevice)
	return nil
}

// DeviceConfig holds configuration for a BLE device (for backward compatibility)
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

	// For simplicity, connect to each device sequentially
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
