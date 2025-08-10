package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinygo.org/x/bluetooth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("🔍 Device Explorer")
		fmt.Println("==================")
		fmt.Println("This tool connects to a specific BLE device and explores its services/characteristics.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Printf("  %s <device_name_or_address>\n", os.Args[0])
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Printf("  %s \"Timeular Tra\"\n", os.Args[0])
		fmt.Printf("  %s \"b566c32a-6f35-262d-5790-dc5777cf683e\"\n", os.Args[0])
		fmt.Println("")
		fmt.Println("💡 Use scan_devices.go first to find available devices")
		os.Exit(1)
	}

	target := os.Args[1]
	fmt.Printf("🔍 Device Explorer - Target: %s\n", target)
	fmt.Println("========================================")
	fmt.Println("")

	// Initialize adapter
	adapter := bluetooth.DefaultAdapter
	fmt.Println("🔌 Enabling BLE adapter...")
	if err := adapter.Enable(); err != nil {
		log.Fatalf("❌ Failed to enable adapter: %v", err)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("✅ BLE adapter enabled")

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Connect to target device
	device, err := connectToTarget(adapter, target)
	if err != nil {
		log.Fatalf("❌ Failed to connect: %v", err)
	}
	defer device.Disconnect()

	fmt.Println("🎉 Connected! Now exploring device...")
	fmt.Println("")

	// Discover all services
	services, err := device.DiscoverServices(nil)
	if err != nil {
		log.Fatalf("❌ Failed to discover services: %v", err)
	}

	fmt.Printf("📋 Found %d services:\n", len(services))
	fmt.Println("")

	// Explore each service
	for i, service := range services {
		fmt.Printf("🔧 Service %d: %s\n", i+1, service.UUID().String())
		fmt.Printf("   %s\n", identifyService(service.UUID()))

		// Discover characteristics for this service
		chars, err := service.DiscoverCharacteristics(nil)
		if err != nil {
			fmt.Printf("   ❌ Failed to discover characteristics: %v\n", err)
			continue
		}

		fmt.Printf("   Found %d characteristics:\n", len(chars))

		for j, char := range chars {
			fmt.Printf("      %d. %s\n", j+1, char.UUID().String())
			fmt.Printf("         %s\n", identifyCharacteristic(char.UUID()))

			// Test characteristic capabilities
			testCharacteristic(&char, i+1, j+1)
		}
		fmt.Println("")
	}

	// Look for Timeular-specific services
	timeularService := findTimeularService(services)
	if timeularService != nil {
		fmt.Println("🎯 TIMEULAR SERVICE DETECTED!")
		fmt.Println("============================")
		exploreTimeularService(timeularService)
		fmt.Println("")
	}

	// Interactive data monitoring
	fmt.Println("📊 Starting interactive data monitoring...")
	fmt.Println("🎲 If this is a Timeular device, try rotating it to different sides!")
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Set up notification monitoring for all characteristics
	monitorAllCharacteristics(services, sigChan)

	// Wait for shutdown
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")
	fmt.Println("🧹 Cleaning up...")

	// Disable all notifications
	for _, service := range services {
		chars, _ := service.DiscoverCharacteristics(nil)
		for _, char := range chars {
			char.EnableNotifications(nil)
		}
	}

	fmt.Println("👋 Exploration complete!")
}

func connectToTarget(adapter *bluetooth.Adapter, target string) (*bluetooth.Device, error) {
	fmt.Printf("🔍 Searching for device: %s\n", target)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	found := make(chan bluetooth.ScanResult, 1)

	go func() {
		err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			name := result.LocalName()
			address := result.Address.String()

			// Check if this matches our target (by name or address)
			if name == target || address == target {
				fmt.Printf("📱 Found target device: '%s' [%s] RSSI: %d\n", name, address, result.RSSI)
				adapter.StopScan()
				found <- result
				return
			}

			// Also check partial matches for unnamed devices
			if target == address || (name == "" && strings.Contains(address, target)) {
				fmt.Printf("📱 Found target device (address match): '%s' [%s] RSSI: %d\n", name, address, result.RSSI)
				adapter.StopScan()
				found <- result
				return
			}
		})
		if err != nil {
			fmt.Printf("❌ Scan error: %v\n", err)
		}
	}()

	var result bluetooth.ScanResult
	select {
	case result = <-found:
		// Found target device
	case <-ctx.Done():
		adapter.StopScan()
		return nil, fmt.Errorf("target device '%s' not found within 30 seconds", target)
	}

	// Connect to device
	fmt.Printf("🔗 Connecting to %s [%s]...\n", target, result.Address.String())
	time.Sleep(500 * time.Millisecond)

	device, err := adapter.Connect(result.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	fmt.Println("✅ Device connected")
	return &device, nil
}

func identifyService(uuid bluetooth.UUID) string {
	knownServices := map[string]string{
		"1800":                                 "Generic Access Service",
		"1801":                                 "Generic Attribute Service",
		"180a":                                 "Device Information Service",
		"180f":                                 "Battery Service",
		"1812":                                 "Human Interface Device Service",
		"c7e70010-c847-11e6-8175-8c89a55d403c": "🎯 TIMEULAR SERVICE",
		"0000180f-0000-1000-8000-00805f9b34fb": "Battery Service",
		"0000180a-0000-1000-8000-00805f9b34fb": "Device Information Service",
	}

	uuidStr := uuid.String()

	// Check full UUID
	if desc, exists := knownServices[uuidStr]; exists {
		return desc
	}

	// Check short UUID (last 4 chars before the standard suffix)
	if len(uuidStr) >= 8 {
		shortUUID := strings.ToLower(uuidStr[4:8])
		if desc, exists := knownServices[shortUUID]; exists {
			return desc
		}
	}

	// Check for Timeular UUID pattern
	if strings.Contains(strings.ToLower(uuidStr), "c7e7") {
		return "🎯 POSSIBLE TIMEULAR SERVICE"
	}

	return "Unknown service"
}

func identifyCharacteristic(uuid bluetooth.UUID) string {
	knownChars := map[string]string{
		"2a00":                                 "Device Name",
		"2a01":                                 "Appearance",
		"2a04":                                 "Peripheral Preferred Connection Parameters",
		"2a19":                                 "Battery Level",
		"2a29":                                 "Manufacturer Name String",
		"2a24":                                 "Model Number String",
		"2a25":                                 "Serial Number String",
		"2a27":                                 "Hardware Revision String",
		"2a26":                                 "Firmware Revision String",
		"2a28":                                 "Software Revision String",
		"c7e70011-c847-11e6-8175-8c89a55d403c": "🎯 TIMEULAR DATA CHARACTERISTIC",
	}

	uuidStr := uuid.String()

	// Check full UUID
	if desc, exists := knownChars[uuidStr]; exists {
		return desc
	}

	// Check short UUID
	if len(uuidStr) >= 8 {
		shortUUID := strings.ToLower(uuidStr[4:8])
		if desc, exists := knownChars[shortUUID]; exists {
			return desc
		}
	}

	// Check for Timeular UUID pattern
	if strings.Contains(strings.ToLower(uuidStr), "c7e7") {
		return "🎯 POSSIBLE TIMEULAR CHARACTERISTIC"
	}

	return "Unknown characteristic"
}

func testCharacteristic(char *bluetooth.DeviceCharacteristic, serviceNum, charNum int) {
	// Test read capability
	canRead := false
	data := make([]byte, 20)
	n, err := char.Read(data)
	if err == nil && n > 0 {
		canRead = true
		data = data[:n]
		fmt.Printf("         📖 Read: [%x] (%d bytes)\n", data, len(data))

		// Try to interpret the data
		if len(data) > 0 {
			interpretData(data, fmt.Sprintf("S%d-C%d", serviceNum, charNum))
		}
	} else if err == nil {
		fmt.Printf("         📖 Read: No data available\n")
	} else {
		fmt.Printf("         📖 Read: Not supported (%v)\n", err)
	}

	// Test notification capability
	canNotify := false
	err = char.EnableNotifications(func(data []byte) {
		// Test callback
	})
	if err == nil {
		canNotify = true
		char.EnableNotifications(nil) // Disable test notification
		fmt.Printf("         🔔 Notifications: Supported\n")
	} else {
		fmt.Printf("         🔔 Notifications: Not supported (%v)\n", err)
	}

	// Summary
	if canRead && canNotify {
		fmt.Printf("         ⭐ Full access (read + notify)\n")
	} else if canRead {
		fmt.Printf("         📚 Read-only access\n")
	} else if canNotify {
		fmt.Printf("         🔔 Notification-only access\n")
	} else {
		fmt.Printf("         ❌ Limited access\n")
	}
}

func interpretData(data []byte, location string) {
	if len(data) == 0 {
		return
	}

	// Check for text data
	if isTextData(data) {
		fmt.Printf("         💬 Text: \"%s\"\n", string(data))
		return
	}

	// Check for potential side data (1-8 range)
	if len(data) == 1 && data[0] >= 1 && data[0] <= 8 {
		fmt.Printf("         🎲 Possible side: %d\n", data[0])
		return
	}

	// Check for potential Timeular sensor data (12 bytes)
	if len(data) == 12 {
		fmt.Printf("         🎯 12-byte data (Timeular format?)\n")
		// Calculate potential side
		sum := 0
		for i, b := range data {
			sum += int(b) * (i + 1)
		}
		side := (sum % 8) + 1
		fmt.Printf("         🎲 Calculated side: %d\n", side)
		return
	}

	// Generic interpretation
	fmt.Printf("         📊 Raw data analysis:\n")
	fmt.Printf("            Length: %d bytes\n", len(data))
	fmt.Printf("            First byte: %d (0x%02x)\n", data[0], data[0])
	if len(data) > 1 {
		fmt.Printf("            Last byte: %d (0x%02x)\n", data[len(data)-1], data[len(data)-1])
	}
}

func isTextData(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

func findTimeularService(services []bluetooth.DeviceService) *bluetooth.DeviceService {
	expectedUUID := bluetooth.NewUUID([16]byte{0xc7, 0xe7, 0x00, 0x10, 0xc8, 0x47, 0x11, 0xe6, 0x81, 0x75, 0x8c, 0x89, 0xa5, 0x5d, 0x40, 0x3c})

	for _, service := range services {
		if service.UUID() == expectedUUID {
			return &service
		}
	}
	return nil
}

func exploreTimeularService(service *bluetooth.DeviceService) {
	chars, err := service.DiscoverCharacteristics(nil)
	if err != nil {
		fmt.Printf("❌ Failed to discover Timeular characteristics: %v\n", err)
		return
	}

	expectedCharUUID := bluetooth.NewUUID([16]byte{0xc7, 0xe7, 0x00, 0x11, 0xc8, 0x47, 0x11, 0xe6, 0x81, 0x75, 0x8c, 0x89, 0xa5, 0x5d, 0x40, 0x3c})

	for _, char := range chars {
		fmt.Printf("🔍 Timeular Characteristic: %s\n", char.UUID().String())

		if char.UUID() == expectedCharUUID {
			fmt.Println("   ✅ This is the expected Timeular data characteristic!")

			// Try to read current data
			data := make([]byte, 20)
			n, err := char.Read(data)
			if err == nil && n > 0 {
				data = data[:n]
				fmt.Printf("   📊 Current data: [%x] (%d bytes)\n", data, len(data))

				if len(data) >= 1 {
					interpretTimeularData(data)
				}
			} else {
				fmt.Printf("   📊 No current data available (%v)\n", err)
			}

			// Test notifications
			fmt.Println("   🔔 Testing notifications...")
			err = char.EnableNotifications(func(data []byte) {
				fmt.Printf("   📡 Notification: [%x] (%d bytes)\n", data, len(data))
				interpretTimeularData(data)
			})

			if err == nil {
				fmt.Println("   ✅ Notifications enabled successfully!")
			} else {
				fmt.Printf("   ❌ Notifications failed: %v\n", err)
			}
		}
	}
}

func interpretTimeularData(data []byte) {
	if len(data) == 0 {
		fmt.Println("      ⚠️ Empty data")
		return
	}

	fmt.Printf("      📊 Timeular Data Analysis:\n")
	fmt.Printf("         Raw: [%x]\n", data)
	fmt.Printf("         Length: %d bytes\n", len(data))
	fmt.Printf("         As integers: %v\n", data)

	// Check for all zeros
	allZero := true
	for _, b := range data {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		fmt.Printf("         💤 Device appears inactive (all zeros)\n")
		return
	}

	// Single byte interpretation
	if len(data) == 1 {
		if data[0] >= 1 && data[0] <= 8 {
			fmt.Printf("         🎲 DETECTED SIDE: %d\n", data[0])
		} else {
			fmt.Printf("         ❓ Unexpected single byte value: %d\n", data[0])
		}
		return
	}

	// 12-byte interpretation (common for sensor data)
	if len(data) == 12 {
		fmt.Printf("         🎯 12-byte sensor data detected\n")

		// Calculate side using different methods
		// Method 1: Simple sum
		sum := 0
		for _, b := range data {
			sum += int(b)
		}
		side1 := (sum % 8) + 1
		fmt.Printf("         🎲 Side (sum method): %d\n", side1)

		// Method 2: Weighted sum
		weightedSum := 0
		for i, b := range data {
			weightedSum += int(b) * (i + 1)
		}
		side2 := (weightedSum % 8) + 1
		fmt.Printf("         🎲 Side (weighted method): %d\n", side2)

		// Method 3: First few bytes
		if len(data) >= 3 {
			side3 := ((int(data[0]) + int(data[1]) + int(data[2])) % 8) + 1
			fmt.Printf("         🎲 Side (first 3 bytes): %d\n", side3)
		}
	}

	fmt.Printf("         ⚡ Active data - device is responding!\n")
}

func monitorAllCharacteristics(services []bluetooth.DeviceService, sigChan chan os.Signal) {
	dataReceived := 0

	// Enable notifications on all characteristics that support it
	for i, service := range services {
		chars, err := service.DiscoverCharacteristics(nil)
		if err != nil {
			continue
		}

		for j, char := range chars {
			err := char.EnableNotifications(func(data []byte) {
				dataReceived++
				timestamp := time.Now().Format("15:04:05.000")
				fmt.Printf("📡 [%s] Data from S%d-C%d: [%x] (%d bytes)\n",
					timestamp, i+1, j+1, data, len(data))

				// Special handling for potential Timeular data
				if len(data) == 1 && data[0] >= 1 && data[0] <= 8 {
					fmt.Printf("   🎲 SIDE CHANGE DETECTED: %d\n", data[0])
				} else if len(data) == 12 {
					fmt.Printf("   🎯 Sensor data - analyzing...\n")
					interpretTimeularData(data)
				}
				fmt.Println("")
			})

			if err == nil {
				fmt.Printf("✅ Monitoring S%d-C%d for notifications\n", i+1, j+1)
			}
		}
	}

	// Status updates
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sigChan:
				return
			case <-ticker.C:
				fmt.Printf("📊 Status: Monitoring active, %d notifications received\n", dataReceived)
				if dataReceived == 0 {
					fmt.Printf("💡 Try interacting with the device (rotate, tap, button press)\n")
				}
				fmt.Println("")
			}
		}
	}()
}
