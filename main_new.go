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

func main() {
	fmt.Println("🖊️  Columbus Video Pen - New Modular Test")
	fmt.Println("==========================================")
	fmt.Println("Testing with simplified connection logic")
	fmt.Println("")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize adapter
	adapter := bluetooth.DefaultAdapter
	fmt.Println("🔌 Enabling BLE adapter...")
	if err := adapter.Enable(); err != nil {
		log.Fatalf("❌ Failed to enable adapter: %v", err)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled")

	// Connect to device
	fmt.Println("🔍 Searching for Columbus Video Pen...")
	device, channel, err := connectToColumbusDevice(adapter)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer device.Disconnect()

	fmt.Println("🎉 Connected and ready!")
	fmt.Println("📝 Tap the Columbus Video Pen on different locations!")
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Listen for signals
	go func() {
		for {
			select {
			case data := <-channel:
				fmt.Printf("🖊️  Signal received: [%x] (length: %d)\n", data, len(data))

				// Validate signal before processing
				if len(data) == 0 {
					fmt.Printf("⚠️  Empty signal received - device may be disconnecting\n")
					continue
				}

				// Extract country from signal (simplified)
				hexStr := fmt.Sprintf("%x", data)
				if len(hexStr) >= 14 {
					countryHex := hexStr[10:14]
					fmt.Printf("📍 Country hex: %s\n", countryHex)
					fmt.Printf("🌍 Country detected from signal!\n")
				} else {
					fmt.Printf("⚠️  Signal too short for country extraction: %s\n", hexStr)
				}

				fmt.Printf("🎯 ACTION: Would trigger HTTP request for detected country\n")
				fmt.Println("")

			case <-sigChan:
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")
	fmt.Println("👋 Goodbye!")
}

func connectToColumbusDevice(adapter *bluetooth.Adapter) (*bluetooth.Device, <-chan []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	found := make(chan bluetooth.ScanResult, 1)
	scanErr := make(chan error, 1)

	go func() {
		err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
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

	var result bluetooth.ScanResult
	select {
	case result = <-found:
		// Device found
	case err := <-scanErr:
		return nil, nil, fmt.Errorf("scan failed: %v", err)
	case <-ctx.Done():
		adapter.StopScan()
		return nil, nil, fmt.Errorf("device not found within timeout")
	}

	// Connect to device
	fmt.Printf("🔗 Connecting to %s...\n", result.Address.String())
	time.Sleep(500 * time.Millisecond) // Brief delay after stopping scan

	device, err := adapter.Connect(result.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("connection failed: %v", err)
	}
	fmt.Println("✅ Device connected")

	// Discover services
	fmt.Println("🔍 Discovering services...")
	services, err := device.DiscoverServices([]bluetooth.UUID{ColumbusServiceUUID})
	if err != nil {
		device.Disconnect()
		return nil, nil, fmt.Errorf("service discovery failed: %v", err)
	}

	if len(services) == 0 {
		device.Disconnect()
		return nil, nil, fmt.Errorf("Nordic UART service not found")
	}

	service := services[0]
	fmt.Printf("✅ Found service: %s\n", service.UUID().String())

	// Discover characteristics
	fmt.Println("🔍 Discovering characteristics...")
	characteristics, err := service.DiscoverCharacteristics([]bluetooth.UUID{ColumbusCharacteristicUUID})
	if err != nil {
		device.Disconnect()
		return nil, nil, fmt.Errorf("characteristic discovery failed: %v", err)
	}

	if len(characteristics) == 0 {
		device.Disconnect()
		return nil, nil, fmt.Errorf("UART TX characteristic not found")
	}

	characteristic := characteristics[0]
	fmt.Printf("✅ Found characteristic: %s\n", characteristic.UUID().String())

	// Set up notifications
	fmt.Println("🔔 Setting up notifications...")
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
		return nil, nil, fmt.Errorf("failed to enable notifications: %v", err)
	}

	fmt.Println("✅ Notifications enabled")
	return &device, channel, nil
}
