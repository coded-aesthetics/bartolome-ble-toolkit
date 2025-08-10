package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

const ColumbusDeviceName = "COLUMBUS Video Pen"

var (
	ColumbusServiceUUID        = bluetooth.ServiceUUIDNordicUART
	ColumbusCharacteristicUUID = bluetooth.CharacteristicUUIDUARTTX
)

// Simple country resolution for demonstration
type Country struct {
	Name       string
	Alpha2Code string
	Region     string
	GlobeHex   string
}

var mockCountries = map[string]*Country{
	"1234": {"United States", "US", "Americas", "1234"},
	"5678": {"Germany", "DE", "Europe", "5678"},
	"9abc": {"Japan", "JP", "Asia", "9abc"},
	"def0": {"Australia", "AU", "Oceania", "def0"},
	"2468": {"United Kingdom", "GB", "Europe", "2468"},
	"1357": {"France", "FR", "Europe", "1357"},
	"8642": {"Canada", "CA", "Americas", "8642"},
	"9753": {"Brazil", "BR", "Americas", "9753"},
}

func resolveCountryFromSignal(signal []byte) (*Country, error) {
	if len(signal) == 0 {
		return nil, fmt.Errorf("empty signal")
	}

	hexStr := fmt.Sprintf("%x", signal)
	if len(hexStr) < 14 {
		return nil, fmt.Errorf("signal too short: %s (length: %d)", hexStr, len(hexStr))
	}

	// Extract country hex (positions 10-13 in hex string)
	countryHex := hexStr[10:14]

	if country, exists := mockCountries[countryHex]; exists {
		return country, nil
	}

	// Return unknown country for codes not in our mock database
	return &Country{
		Name:       fmt.Sprintf("Unknown Country (%s)", countryHex),
		Alpha2Code: "XX",
		Region:     "Unknown",
		GlobeHex:   countryHex,
	}, nil
}

func main() {
	fmt.Println("🖊️  Columbus Video Pen - Working Example")
	fmt.Println("========================================")
	fmt.Println("This example demonstrates reliable connection to Columbus Video Pen")
	fmt.Println("with country detection from pen signals.")
	fmt.Println("")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize and connect to device
	fmt.Println("🚀 Initializing connection...")
	device, channel, err := connectToColumbus()
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer device.Disconnect()

	fmt.Println("🎉 Columbus Video Pen connected and ready!")
	fmt.Println("📝 Tap your Columbus Video Pen on different locations to detect countries!")
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Track statistics
	var signalsReceived int
	startTime := time.Now()

	// Listen for signals
	go func() {
		for {
			select {
			case data := <-channel:
				signalsReceived++
				fmt.Printf("🖊️  Signal #%d received: [%x] (length: %d)\n", signalsReceived, data, len(data))

				// Validate and process signal
				if len(data) == 0 {
					fmt.Println("⚠️  Empty signal received - device may be disconnecting")
					continue
				}

				// Resolve country from signal
				country, err := resolveCountryFromSignal(data)
				if err != nil {
					fmt.Printf("❌ Country resolution failed: %v\n", err)
					continue
				}

				// Display country information
				fmt.Printf("🌍 Country: %s (%s)\n", country.Name, country.Alpha2Code)
				fmt.Printf("🗺️  Region: %s\n", country.Region)
				fmt.Printf("🔢 Country Code: %s\n", country.GlobeHex)

				// Simulate action trigger
				fmt.Printf("🎯 ACTION: Triggering request for %s\n", country.Name)
				fmt.Printf("📊 Session stats: %d signals in %.1f minutes\n",
					signalsReceived, time.Since(startTime).Minutes())
				fmt.Println("")

			case <-sigChan:
				return
			}
		}
	}()

	// Status updates every 30 seconds
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sessionTime := time.Since(startTime)
				fmt.Printf("📊 Status: Connected for %.1f minutes, %d signals received\n",
					sessionTime.Minutes(), signalsReceived)
			case <-sigChan:
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")

	// Display final statistics
	sessionTime := time.Since(startTime)
	fmt.Printf("📊 Final Statistics:\n")
	fmt.Printf("   Session duration: %.1f minutes\n", sessionTime.Minutes())
	fmt.Printf("   Total signals received: %d\n", signalsReceived)
	if sessionTime.Minutes() > 0 {
		signalsPerMinute := float64(signalsReceived) / sessionTime.Minutes()
		fmt.Printf("   Average signals per minute: %.1f\n", signalsPerMinute)
	}

	fmt.Println("🧹 Cleaning up...")
	fmt.Println("👋 Thanks for using the Columbus Video Pen!")
}

func connectToColumbus() (*bluetooth.Device, <-chan []byte, error) {
	adapter := bluetooth.DefaultAdapter

	// Enable BLE adapter
	fmt.Println("🔌 Enabling BLE adapter...")
	if err := adapter.Enable(); err != nil {
		return nil, nil, fmt.Errorf("failed to enable adapter: %v", err)
	}

	// Give macOS time to initialize properly
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled")

	// Scan for device
	fmt.Printf("🔍 Scanning for %s...\n", ColumbusDeviceName)
	result, err := scanForDevice(adapter)
	if err != nil {
		return nil, nil, err
	}

	// Connect to device
	fmt.Printf("🔗 Connecting to %s [%s]...\n", ColumbusDeviceName, result.Address.String())
	device, err := connectAndSetup(adapter, result)
	if err != nil {
		return nil, nil, err
	}

	// Setup notifications
	channel, err := setupNotifications(device)
	if err != nil {
		device.Disconnect()
		return nil, nil, err
	}

	return device, channel, nil
}

func scanForDevice(adapter *bluetooth.Adapter) (bluetooth.ScanResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	found := make(chan bluetooth.ScanResult, 1)
	scanErr := make(chan error, 1)

	go func() {
		err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.LocalName() == ColumbusDeviceName {
				fmt.Printf("📱 Found %s [%s] RSSI: %d\n",
					ColumbusDeviceName, result.Address.String(), result.RSSI)
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
		adapter.StopScan()
		return bluetooth.ScanResult{}, fmt.Errorf("device not found within 30 seconds")
	}
}

func connectAndSetup(adapter *bluetooth.Adapter, result bluetooth.ScanResult) (*bluetooth.Device, error) {
	// Brief delay after stopping scan (important for macOS)
	time.Sleep(500 * time.Millisecond)

	// Connect to device
	device, err := adapter.Connect(result.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	fmt.Println("✅ Device connected")

	// Discover services
	fmt.Println("🔍 Discovering services...")
	services, err := device.DiscoverServices([]bluetooth.UUID{ColumbusServiceUUID})
	if err != nil {
		return nil, fmt.Errorf("service discovery failed: %v", err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("Nordic UART service not found")
	}

	service := services[0]
	fmt.Printf("✅ Found Nordic UART service: %s\n", service.UUID().String())

	// Discover characteristics
	fmt.Println("🔍 Discovering characteristics...")
	characteristics, err := service.DiscoverCharacteristics([]bluetooth.UUID{ColumbusCharacteristicUUID})
	if err != nil {
		return nil, fmt.Errorf("characteristic discovery failed: %v", err)
	}

	if len(characteristics) == 0 {
		return nil, fmt.Errorf("UART TX characteristic not found")
	}

	fmt.Printf("✅ Found UART TX characteristic: %s\n", characteristics[0].UUID().String())
	return &device, nil
}

func setupNotifications(device *bluetooth.Device) (<-chan []byte, error) {
	// Get the characteristic again for notifications
	services, err := device.DiscoverServices([]bluetooth.UUID{ColumbusServiceUUID})
	if err != nil {
		return nil, err
	}

	characteristics, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{ColumbusCharacteristicUUID})
	if err != nil {
		return nil, err
	}

	characteristic := characteristics[0]

	// Setup notification channel
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
		return nil, fmt.Errorf("failed to enable notifications: %v", err)
	}

	fmt.Println("✅ Notifications enabled")
	return channel, nil
}
