package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/ble"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/columbus"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/countries"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/timeular"
)

func main() {
	fmt.Println("🚀 Full Bartolome BLE Setup Example")
	fmt.Println("===================================")
	fmt.Println("This example demonstrates connecting to all supported devices:")
	fmt.Println("- Columbus Video Pen")
	fmt.Println("- Timeular Tracker 1 (custom name)")
	fmt.Println("- Timeular Tracker 2 (custom name)")
	fmt.Println("")

	// Create devices with custom configurations
	columbusDevice := columbus.NewDevice()

	// Create two separate Timeular devices with custom names
	timeularDevice1 := timeular.NewDeviceWithName("Timeular Tracker 1")
	timeularDevice2 := timeular.NewDeviceWithName("Timeular Tracker 2")

	// You can also use different polling intervals for each device
	timeularDevice1.SetPollInterval(500 * timeular.DefaultPollInterval)  // 500ms polling
	timeularDevice2.SetPollInterval(1000 * timeular.DefaultPollInterval) // 1s polling

	// Create a BLE manager
	manager := ble.NewManager()

	// Track device states
	var (
		currentCountry *countries.Country
		timeularSide1  byte = 1
		timeularSide2  byte = 1
	)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up Columbus device signal handler
	columbusDevice.OnSignal(func(signal []byte) error {
		fmt.Printf("🖊️  Columbus signal: [%x] (length: %d)\n", signal, len(signal))

		// Extract and resolve country
		countryHex, err := columbus.SignalToCountryHex(signal)
		if err != nil {
			fmt.Printf("⚠️  Could not extract country: %v\n", err)
			return nil
		}

		country, err := countries.ResolveFromHex(countryHex)
		if err != nil {
			fmt.Printf("❌ Could not resolve country: %v\n", err)
			return nil
		}

		currentCountry = country
		fmt.Printf("🌍 Country: %s (%s)\n", country.Name, country.Alpha2Code)

		// Trigger action with current state
		triggerAction(country.Name, timeularSide1, timeularSide2)
		return nil
	})

	// Set up Timeular device 1 handlers
	timeularDevice1.OnSideChange(func(deviceName string, side byte) error {
		fmt.Printf("🎲 %s side changed: %d\n", deviceName, side)
		timeularSide1 = side

		if currentCountry != nil {
			triggerAction(currentCountry.Name, timeularSide1, timeularSide2)
		}
		return nil
	})

	// Optional: Handle raw data from Timeular device 1
	timeularDevice1.OnData(func(deviceName string, data []byte) error {
		fmt.Printf("📊 %s raw data: [%x]\n", deviceName, data)
		return nil
	})

	// Set up Timeular device 2 handlers
	timeularDevice2.OnSideChange(func(deviceName string, side byte) error {
		fmt.Printf("🎲 %s side changed: %d\n", deviceName, side)
		timeularSide2 = side

		if currentCountry != nil {
			triggerAction(currentCountry.Name, timeularSide1, timeularSide2)
		}
		return nil
	})

	// Optional: Handle raw data from Timeular device 2
	timeularDevice2.OnData(func(deviceName string, data []byte) error {
		fmt.Printf("📊 %s raw data: [%x]\n", deviceName, data)
		return nil
	})

	// Set up disconnect handler
	manager.SetDisconnectHandler(func(deviceName, address string, err error) {
		fmt.Printf("⚠️  Device %s [%s] disconnected: %v\n", deviceName, address, err)
		fmt.Println("🔄 Will attempt to reconnect...")

		// Reset device state on disconnect
		switch deviceName {
		case "Timeular Tracker 1":
			timeularDevice1.Reset()
		case "Timeular Tracker 2":
			timeularDevice2.Reset()
		}
	})

	// Configure all devices for BLE manager
	deviceConfigs := []ble.DeviceConfig{
		{
			Name:               columbusDevice.GetName(),
			ServiceUUID:        columbusDevice.GetServiceUUID(),
			CharacteristicUUID: columbusDevice.GetCharacteristicUUID(),
			NotificationHandler: func(deviceName string, data []byte) error {
				return columbusDevice.ProcessNotification(deviceName, data)
			},
		},
		{
			Name:               timeularDevice1.GetName(),
			ServiceUUID:        timeularDevice1.GetServiceUUID(),
			CharacteristicUUID: timeularDevice1.GetCharacteristicUUID(),
			NotificationHandler: func(deviceName string, data []byte) error {
				return timeularDevice1.ProcessNotification(deviceName, data)
			},
		},
		{
			Name:               timeularDevice2.GetName(),
			ServiceUUID:        timeularDevice2.GetServiceUUID(),
			CharacteristicUUID: timeularDevice2.GetCharacteristicUUID(),
			NotificationHandler: func(deviceName string, data []byte) error {
				return timeularDevice2.ProcessNotification(deviceName, data)
			},
		},
	}

	// Start connecting to devices
	fmt.Println("🔍 Searching for all BLE devices...")
	fmt.Println("📱 Make sure your devices are powered on and nearby!")
	if err := manager.ConnectDevices(deviceConfigs); err != nil {
		log.Fatalf("❌ Failed to start device connection: %v", err)
	}

	fmt.Println("✅ Connection process started")
	fmt.Println("📝 Instructions:")
	fmt.Println("   - Select a country with the Columbus video pen")
	fmt.Println("   - Rotate the Timeular devices to different sides")
	fmt.Println("   - Watch as the system combines all inputs!")
	fmt.Printf("   - Timeular devices support sides 1-%d\n", timeular.GetSupportedSides())
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Display current status periodically
	go func() {
		for {
			select {
			case <-sigChan:
				return
			default:
				connectedDevices := manager.GetConnectedDevices()
				if len(connectedDevices) > 0 {
					fmt.Printf("📊 Status - Connected devices: %d, Timeular 1 side: %d, Timeular 2 side: %d\n",
						len(connectedDevices), timeularDevice1.GetCurrentSide(), timeularDevice2.GetCurrentSide())
				}

				// Sleep for 30 seconds before next status update
				for i := 0; i < 30; i++ {
					select {
					case <-sigChan:
						return
					default:
						time.Sleep(1 * time.Second)
					}
				}
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")

	// Stop Timeular devices
	fmt.Println("🛑 Stopping Timeular devices...")
	timeularDevice1.Stop()
	timeularDevice2.Stop()

	// Clean shutdown
	fmt.Println("🧹 Cleaning up BLE connections...")
	if err := manager.Close(); err != nil {
		fmt.Printf("⚠️  Error during shutdown: %v\n", err)
	}

	fmt.Println("👋 Goodbye!")
}

// triggerAction simulates the action that would be triggered by the combined device inputs
func triggerAction(country string, category1, category2 byte) {
	fmt.Printf("🎯 ACTION TRIGGERED!\n")
	fmt.Printf("   Country: %s\n", country)
	fmt.Printf("   Category 1 (Timeular 1): %d\n", category1)
	fmt.Printf("   Category 2 (Timeular 2): %d\n", category2)
	fmt.Printf("   → This would trigger your custom action (e.g., HTTP request)\n")

	// Validate sides
	if !timeular.IsValidSide(category1) {
		fmt.Printf("   ⚠️  Invalid side for Timeular 1: %d\n", category1)
	}
	if !timeular.IsValidSide(category2) {
		fmt.Printf("   ⚠️  Invalid side for Timeular 2: %d\n", category2)
	}

	fmt.Println("")
}
