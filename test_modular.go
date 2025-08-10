package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

// Columbus Video Pen constants
const ColumbusDeviceName = "COLUMBUS Video Pen"

var (
	ColumbusServiceUUID        = bluetooth.ServiceUUIDNordicUART
	ColumbusCharacteristicUUID = bluetooth.CharacteristicUUIDUARTTX
)

// Simple BLE Manager
type SimpleBLEManager struct {
	adapter   *bluetooth.Adapter
	connected bool
	device    *bluetooth.Device
	channel   chan []byte
	mu        sync.RWMutex
}

func NewSimpleBLEManager() *SimpleBLEManager {
	return &SimpleBLEManager{
		adapter: bluetooth.DefaultAdapter,
		channel: make(chan []byte, 10),
	}
}

func (m *SimpleBLEManager) Connect() error {
	fmt.Println("🔌 Enabling BLE adapter...")
	if err := m.adapter.Enable(); err != nil {
		return fmt.Errorf("failed to enable adapter: %v", err)
	}

	// Give macOS time to initialize
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled")

	return m.scanAndConnect()
}

func (m *SimpleBLEManager) scanAndConnect() error {
	fmt.Println("🔍 Scanning for Columbus Video Pen...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	found := make(chan bluetooth.ScanResult, 1)
	scanErr := make(chan error, 1)

	go func() {
		err := m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.LocalName() == ColumbusDeviceName {
				fmt.Printf("📱 Found %s [%s]\n", ColumbusDeviceName, result.Address.String())
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
		return m.connectToDevice(result)
	case err := <-scanErr:
		return fmt.Errorf("scan failed: %v", err)
	case <-ctx.Done():
		m.adapter.StopScan()
		return fmt.Errorf("device not found within timeout")
	}
}

func (m *SimpleBLEManager) connectToDevice(result bluetooth.ScanResult) error {
	fmt.Printf("🔗 Connecting to %s...\n", result.Address.String())

	// Brief delay after stopping scan
	time.Sleep(500 * time.Millisecond)

	device, err := m.adapter.Connect(result.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}

	fmt.Println("✅ Device connected")

	// Discover services
	fmt.Println("🔍 Discovering services...")
	services, err := device.DiscoverServices([]bluetooth.UUID{ColumbusServiceUUID})
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("service discovery failed: %v", err)
	}

	if len(services) == 0 {
		device.Disconnect()
		return fmt.Errorf("Nordic UART service not found")
	}

	service := services[0]
	fmt.Printf("✅ Found service: %s\n", service.UUID().String())

	// Discover characteristics
	fmt.Println("🔍 Discovering characteristics...")
	characteristics, err := service.DiscoverCharacteristics([]bluetooth.UUID{ColumbusCharacteristicUUID})
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("characteristic discovery failed: %v", err)
	}

	if len(characteristics) == 0 {
		device.Disconnect()
		return fmt.Errorf("UART TX characteristic not found")
	}

	characteristic := characteristics[0]
	fmt.Printf("✅ Found characteristic: %s\n", characteristic.UUID().String())

	// Enable notifications
	fmt.Println("🔔 Setting up notifications...")
	err = characteristic.EnableNotifications(func(data []byte) {
		select {
		case m.channel <- data:
		default:
			// Channel full, drop data
			fmt.Println("⚠️ Dropped notification - channel full")
		}
	})

	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to enable notifications: %v", err)
	}

	fmt.Println("✅ Notifications enabled")

	m.mu.Lock()
	m.connected = true
	m.device = &device
	m.mu.Unlock()

	fmt.Println("🎉 Columbus Video Pen connected and ready!")
	return nil
}

func (m *SimpleBLEManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *SimpleBLEManager) GetDataChannel() <-chan []byte {
	return m.channel
}

func (m *SimpleBLEManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.device != nil {
		m.device.Disconnect()
		m.device = nil
	}
	m.connected = false

	return nil
}

// Country resolution (simplified)
type Country struct {
	Name       string
	Alpha2Code string
	Region     string
}

func ResolveCountryFromSignal(signal []byte) (*Country, error) {
	if len(signal) == 0 {
		return nil, fmt.Errorf("empty signal")
	}

	hexStr := fmt.Sprintf("%x", signal)
	if len(hexStr) < 14 {
		return nil, fmt.Errorf("signal too short: %s", hexStr)
	}

	// Extract country hex (simplified)
	countryHex := hexStr[10:14]

	// Mock country resolution for testing
	countries := map[string]*Country{
		"1234": {"United States", "US", "Americas"},
		"5678": {"Germany", "DE", "Europe"},
		"9abc": {"Japan", "JP", "Asia"},
		"def0": {"Australia", "AU", "Oceania"},
	}

	if country, exists := countries[countryHex]; exists {
		return country, nil
	}

	// Return a default country for unknown codes
	return &Country{
		Name:       fmt.Sprintf("Unknown Country (%s)", countryHex),
		Alpha2Code: "XX",
		Region:     "Unknown",
	}, nil
}

func main() {
	fmt.Println("🖊️ Modular Columbus Video Pen Test")
	fmt.Println("===================================")
	fmt.Println("Testing the new modular approach with working connection code")
	fmt.Println("")

	// Create manager
	manager := NewSimpleBLEManager()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Connect to device
	fmt.Println("🚀 Starting connection process...")
	if err := manager.Connect(); err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}

	fmt.Println("📡 Listening for signals...")
	fmt.Println("🖊️ Tap your Columbus Video Pen on different locations!")
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Listen for signals
	go func() {
		for {
			select {
			case data := <-manager.GetDataChannel():
				fmt.Printf("🖊️ Signal received: [%x] (length: %d)\n", data, len(data))

				// Validate signal
				if len(data) == 0 {
					fmt.Println("⚠️ Empty signal - device may be disconnecting")
					continue
				}

				// Resolve country
				country, err := ResolveCountryFromSignal(data)
				if err != nil {
					fmt.Printf("❌ Country resolution failed: %v\n", err)
					continue
				}

				fmt.Printf("🌍 Country: %s (%s)\n", country.Name, country.Alpha2Code)
				fmt.Printf("🗺️ Region: %s\n", country.Region)
				fmt.Printf("🎯 Action: Would trigger HTTP request for %s\n", country.Name)
				fmt.Println("")

			case <-sigChan:
				return
			}
		}
	}()

	// Status monitoring
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if manager.IsConnected() {
					fmt.Println("📊 Status: Device connected and listening")
				} else {
					fmt.Println("📊 Status: Device not connected")
				}
			case <-sigChan:
				return
			}
		}
	}()

	// Wait for shutdown
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")

	// Clean shutdown
	fmt.Println("🧹 Cleaning up...")
	if err := manager.Close(); err != nil {
		fmt.Printf("⚠️ Error during cleanup: %v\n", err)
	}

	fmt.Println("👋 Goodbye!")
}
