// Package ble provides a high-level interface for managing Bluetooth Low Energy devices
// with automatic connection, reconnection, and device management capabilities.
package ble

import (
	"context"
	"fmt"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// DeviceConfig defines the configuration for a BLE device to connect to
type DeviceConfig struct {
	Name                string                                     // Human-readable device name
	ServiceUUID         bluetooth.UUID                             // Primary service UUID to look for
	CharacteristicUUID  bluetooth.UUID                             // Characteristic UUID to subscribe to
	NotificationHandler func(deviceName string, data []byte) error // Handler for incoming notifications
}

// ConnectedDevice represents a connected BLE device
type ConnectedDevice struct {
	Name           string
	Address        bluetooth.Address
	Device         *bluetooth.Device
	Service        *bluetooth.DeviceService
	Characteristic *bluetooth.DeviceCharacteristic
	Channel        chan []byte
	cancel         func() // Function to disable notifications
}

// Manager handles BLE device connections and reconnections
type Manager struct {
	devices           map[string]*ConnectedDevice
	configs           []DeviceConfig
	disconnectHandler func(deviceName string, address string, err error)
	stopChannel       chan bool
	adapter           *bluetooth.Adapter
	mu                sync.RWMutex
	running           bool
}

// NewManager creates a new BLE manager instance
func NewManager() *Manager {
	return &Manager{
		devices:     make(map[string]*ConnectedDevice),
		stopChannel: make(chan bool, 1),
		adapter:     bluetooth.DefaultAdapter,
	}
}

// SetDisconnectHandler sets the callback function for device disconnections
func (m *Manager) SetDisconnectHandler(handler func(deviceName string, address string, err error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectHandler = handler
}

// ConnectDevices attempts to connect to all specified devices with automatic reconnection
func (m *Manager) ConnectDevices(configs []DeviceConfig) error {
	m.mu.Lock()
	m.configs = make([]DeviceConfig, len(configs))
	copy(m.configs, configs)
	m.running = true
	m.mu.Unlock()

	// Enable BLE adapter
	if err := m.enableAdapter(); err != nil {
		return fmt.Errorf("failed to enable BLE adapter: %v", err)
	}

	// Start connection management in background
	go m.connectionManager()

	return nil
}

// GetConnectedDevices returns a map of currently connected devices
func (m *Manager) GetConnectedDevices() map[string]*ConnectedDevice {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ConnectedDevice)
	for name, device := range m.devices {
		result[name] = device
	}
	return result
}

// IsConnected checks if a specific device is connected
func (m *Manager) IsConnected(deviceName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.devices[deviceName]
	return exists
}

// Close stops the manager and disconnects all devices
func (m *Manager) Close() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	// Signal stop
	select {
	case m.stopChannel <- true:
	default:
	}

	// Disconnect all devices
	m.mu.Lock()
	for _, device := range m.devices {
		m.disconnectDevice(device)
	}
	m.devices = make(map[string]*ConnectedDevice)
	m.mu.Unlock()

	return nil
}

// enableAdapter enables the BLE adapter with proper synchronization
func (m *Manager) enableAdapter() error {
	fmt.Println("🔌 Enabling BLE adapter...")

	if err := m.adapter.Enable(); err != nil {
		return fmt.Errorf("could not enable BLE adapter: %v", err)
	}

	// Give macOS time to initialize properly
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled successfully")

	return nil
}

// connectionManager handles the main connection/reconnection loop
func (m *Manager) connectionManager() {
	for {
		select {
		case <-m.stopChannel:
			fmt.Println("🛑 BLE manager stopped")
			return
		default:
		}

		fmt.Println("🔄 Starting device discovery...")

		if err := m.discoverAndConnectDevices(); err != nil {
			fmt.Printf("❌ Connection attempt failed: %v\n", err)
			fmt.Println("⏰ Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}

		fmt.Println("✅ Device discovery completed successfully")

		// Monitor for disconnections
		disconnectChannel := m.setupDisconnectMonitoring()

		select {
		case <-m.stopChannel:
			return
		case err := <-disconnectChannel:
			fmt.Printf("⚠️  Device disconnected: %v\n", err)
			time.Sleep(3 * time.Second) // Brief delay before reconnection
		}
	}
}

// discoverAndConnectDevices scans for and connects to configured devices
func (m *Manager) discoverAndConnectDevices() error {
	fmt.Println("🔍 Starting device discovery process...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	devicesFound := make(chan *ConnectedDevice, len(m.configs))
	errChannel := make(chan error, 1)
	scanComplete := make(chan bool, 1)

	go func() {
		defer func() {
			close(devicesFound)
			scanComplete <- true
		}()

		fmt.Println("📡 Starting BLE scan...")

		if err := m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			device := m.processDiscoveredDevice(result)
			if device != nil {
				select {
				case devicesFound <- device:
				default:
					// Channel full, but device was processed
				}

				// Check if we found all devices
				m.mu.RLock()
				connectedCount := len(m.devices)
				targetCount := len(m.configs)
				m.mu.RUnlock()

				if connectedCount >= targetCount {
					fmt.Println("📱 All devices found, stopping scan...")
					adapter.StopScan()
					return
				}
			}
		}); err != nil {
			select {
			case errChannel <- fmt.Errorf("scan failed: %v", err):
			default:
			}
		}
	}()

	// Collect devices or timeout
	for {
		select {
		case device, ok := <-devicesFound:
			if !ok {
				// Channel closed, scan completed
				m.mu.RLock()
				connectedCount := len(m.devices)
				targetCount := len(m.configs)
				m.mu.RUnlock()

				if connectedCount == 0 {
					return fmt.Errorf("no devices found")
				}

				fmt.Printf("📱 Connected to %d/%d devices\n", connectedCount, targetCount)
				return nil
			}

			if device != nil {
				m.mu.Lock()
				m.devices[device.Name] = device
				fmt.Printf("✅ Connected to %s [%s]\n", device.Name, device.Address.String())

				// Check if we have all devices
				connectedCount := len(m.devices)
				targetCount := len(m.configs)
				m.mu.Unlock()

				if connectedCount >= targetCount {
					m.adapter.StopScan()
					fmt.Printf("🎉 All %d devices connected!\n", connectedCount)
					return nil
				}
			}
		case err := <-errChannel:
			m.adapter.StopScan()
			return err
		case <-ctx.Done():
			m.adapter.StopScan()

			m.mu.RLock()
			connectedCount := len(m.devices)
			targetCount := len(m.configs)
			m.mu.RUnlock()

			if connectedCount == 0 {
				return fmt.Errorf("no devices found within timeout")
			}

			fmt.Printf("📱 Timeout reached - connected to %d/%d devices\n", connectedCount, targetCount)
			return nil
		}
	}
}

// processDiscoveredDevice processes a discovered device and attempts connection
func (m *Manager) processDiscoveredDevice(result bluetooth.ScanResult) *ConnectedDevice {
	config := m.findDeviceConfig(result)
	if config == nil {
		return nil
	}

	// Check if already connected
	m.mu.RLock()
	if _, exists := m.devices[config.Name]; exists {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	fmt.Printf("📱 Found %s, connecting...\n", config.Name)

	// Stop scanning to connect (macOS requirement)
	fmt.Println("⏸️  Stopping scan for connection...")
	m.adapter.StopScan()

	// Brief delay to ensure scan is stopped
	time.Sleep(500 * time.Millisecond)

	fmt.Printf("🔗 Attempting to connect to %s...\n", config.Name)
	device, err := m.connectToDevice(result, *config)
	if err != nil {
		fmt.Printf("❌ Failed to connect to %s: %v\n", config.Name, err)
		// Try to restart scan for remaining devices
		go func() {
			time.Sleep(2 * time.Second)
			// Don't restart scan here as it causes issues
		}()
		return nil
	}
	fmt.Printf("✅ Successfully connected to %s\n", config.Name)

	return device
}

// findDeviceConfig finds the configuration for a discovered device
func (m *Manager) findDeviceConfig(result bluetooth.ScanResult) *DeviceConfig {
	deviceName := result.LocalName()

	for _, config := range m.configs {
		// Try matching by service UUID first
		if result.AdvertisementPayload.HasServiceUUID(config.ServiceUUID) {
			return &config
		}

		// Fallback to name matching for macOS compatibility
		if deviceName != "" && config.Name == deviceName {
			return &config
		}
	}

	return nil
}

// connectToDevice establishes connection to a specific device
func (m *Manager) connectToDevice(result bluetooth.ScanResult, config DeviceConfig) (*ConnectedDevice, error) {
	// Connect to device
	fmt.Printf("🔌 Connecting to device at address %s...\n", result.Address.String())
	device, err := m.adapter.Connect(result.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	fmt.Printf("🔗 Device connection established\n")

	// Discover services
	fmt.Printf("🔍 Discovering services for %s...\n", config.Name)
	services, err := device.DiscoverServices([]bluetooth.UUID{config.ServiceUUID})
	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("service discovery failed: %v", err)
	}
	fmt.Printf("📋 Found %d services\n", len(services))

	if len(services) == 0 {
		device.Disconnect()
		return nil, fmt.Errorf("service not found")
	}

	service := services[0]

	// Discover characteristics
	fmt.Printf("🔍 Discovering characteristics for %s...\n", config.Name)
	characteristics, err := service.DiscoverCharacteristics([]bluetooth.UUID{config.CharacteristicUUID})
	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("characteristic discovery failed: %v", err)
	}
	fmt.Printf("📋 Found %d characteristics\n", len(characteristics))

	if len(characteristics) == 0 {
		device.Disconnect()
		return nil, fmt.Errorf("characteristic not found")
	}

	characteristic := characteristics[0]

	// Setup notifications
	fmt.Printf("🔔 Setting up notifications for %s...\n", config.Name)
	channel := make(chan []byte, 10)
	err = characteristic.EnableNotifications(func(data []byte) {
		select {
		case channel <- data:
		default:
			// Channel full, drop data
		}
	})

	if err != nil {
		device.Disconnect()
		return nil, fmt.Errorf("failed to enable notifications: %v", err)
	}
	fmt.Printf("✅ Notifications enabled for %s\n", config.Name)

	connectedDevice := &ConnectedDevice{
		Name:           config.Name,
		Address:        result.Address,
		Device:         &device,
		Service:        &service,
		Characteristic: &characteristic,
		Channel:        channel,
		cancel: func() {
			characteristic.EnableNotifications(nil)
		},
	}

	// Start notification handler
	go m.handleNotifications(connectedDevice, config.NotificationHandler)

	return connectedDevice, nil
}

// handleNotifications processes incoming notifications for a device
func (m *Manager) handleNotifications(device *ConnectedDevice, handler func(string, []byte) error) {
	for data := range device.Channel {
		if handler != nil {
			if err := handler(device.Name, data); err != nil {
				fmt.Printf("⚠️ Notification handler error for %s: %v\n", device.Name, err)
			}
		}
	}
}

// setupDisconnectMonitoring sets up monitoring for device disconnections
func (m *Manager) setupDisconnectMonitoring() chan error {
	disconnectChannel := make(chan error, 1)

	m.adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		if !connected {
			m.mu.RLock()
			var disconnectedDevice *ConnectedDevice
			for _, d := range m.devices {
				if d.Address.String() == device.Address.String() {
					disconnectedDevice = d
					break
				}
			}
			m.mu.RUnlock()

			if disconnectedDevice != nil {
				// Remove from connected devices
				m.mu.Lock()
				delete(m.devices, disconnectedDevice.Name)
				m.mu.Unlock()

				// Clean up device
				m.disconnectDevice(disconnectedDevice)

				// Notify disconnect handler
				if m.disconnectHandler != nil {
					m.disconnectHandler(disconnectedDevice.Name, disconnectedDevice.Address.String(), fmt.Errorf("device disconnected"))
				}

				disconnectChannel <- fmt.Errorf("%s [%s] disconnected", disconnectedDevice.Name, disconnectedDevice.Address.String())
			}
		}
	})

	return disconnectChannel
}

// disconnectDevice cleans up a connected device
func (m *Manager) disconnectDevice(device *ConnectedDevice) {
	if device.cancel != nil {
		device.cancel()
	}
	if device.Device != nil {
		device.Device.Disconnect()
	}
}
