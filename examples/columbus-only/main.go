package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/ble"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/columbus"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/countries"
)

func main() {
	fmt.Println("🖊️  Columbus Video Pen Example")
	fmt.Println("=============================")
	fmt.Println("This example demonstrates connecting to and receiving signals from a Columbus Video Pen.")
	fmt.Println("")

	// Create a Columbus device
	columbusDevice := columbus.NewDevice()

	// Create a BLE manager
	manager := ble.NewManager()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up Columbus device signal handler
	columbusDevice.OnSignal(func(signal []byte) error {
		fmt.Printf("🖊️  Signal received: [%x] (length: %d)\n", signal, len(signal))

		// Validate signal before processing
		if len(signal) == 0 {
			fmt.Printf("⚠️  Empty signal received - device may be disconnecting\n")
			return nil
		}

		// Extract country from signal
		countryHex, err := columbus.SignalToCountryHex(signal)
		if err != nil {
			fmt.Printf("⚠️  Could not extract country hex: %v\n", err)
			return nil
		}

		fmt.Printf("📍 Country hex: %s\n", countryHex)

		// Resolve country
		country, err := countries.ResolveFromHex(countryHex)
		if err != nil {
			fmt.Printf("❌ Could not resolve country: %v\n", err)
			return nil
		}

		fmt.Printf("🌍 Country: %s (%s)\n", country.Name, country.Alpha2Code)
		fmt.Printf("🗺️  Region: %s\n", country.Region)
		if country.SubRegion != "" {
			fmt.Printf("📍 Sub-region: %s\n", country.SubRegion)
		}

		fmt.Println("")

		return nil
	})

	// Set up disconnect handler
	manager.SetDisconnectHandler(func(deviceName, address string, err error) {
		fmt.Printf("⚠️  Device %s [%s] disconnected: %v\n", deviceName, address, err)
		fmt.Println("🔄 Will attempt to reconnect...")
	})

	// Configure device for BLE manager
	deviceConfig := ble.DeviceConfig{
		Name:               columbusDevice.GetName(),
		ServiceUUID:        columbusDevice.GetServiceUUID(),
		CharacteristicUUID: columbusDevice.GetCharacteristicUUID(),
		NotificationHandler: func(deviceName string, data []byte) error {
			return columbusDevice.ProcessNotification(deviceName, data)
		},
	}

	// Start connecting to devices
	fmt.Println("🔍 Searching for Columbus Video Pen...")
	fmt.Println("📱 Make sure your Columbus Video Pen is turned on and nearby!")
	if err := manager.ConnectDevices([]ble.DeviceConfig{deviceConfig}); err != nil {
		log.Fatalf("❌ Failed to connect to device: %v", err)
	}

	fmt.Println("✅ Connection successful!")
	fmt.Println("📝 Select a country with the Columbus video pen to see country detection in action!")
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")

	// Clean shutdown
	fmt.Println("🧹 Cleaning up connections...")
	if err := manager.Close(); err != nil {
		fmt.Printf("⚠️  Error during shutdown: %v\n", err)
	}

	fmt.Println("👋 Goodbye!")
}
