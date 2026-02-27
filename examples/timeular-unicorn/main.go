package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/ble"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/timeular"
)

func main() {
	fmt.Println("🎲 Timeular Tracker Example")
	fmt.Println("===========================")
	fmt.Println("This example demonstrates connecting to a single Timeular tracker device.")
	fmt.Println("Rotate your Timeular device to see side changes in real-time!")
	fmt.Println("")

	// Create a single Timeular device with custom configuration
	timeularDevice := timeular.NewDeviceWithConfig(timeular.Config{
		Name:         "Timeular Tra",
		PollInterval: 500 * time.Millisecond, // Poll every 500ms for faster response
	})

	// Create a BLE manager
	manager := ble.NewManager()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Set up side change handler
	timeularDevice.OnSideChange(func(deviceName string, side byte) error {
		// Validate the side
		if !timeular.IsValidSide(side) {
			fmt.Printf("⚠️  Warning: Invalid side detected: %d\n", side)
			return fmt.Errorf("invalid side: %d", side)
		}

		response, err := http.Get(fmt.Sprintf("http://192.168.0.185/?num=%d", side))
		if err != nil {
			fmt.Printf("Error making request: %v\n", err)
		}
		defer response.Body.Close()

		return nil
	})

	// Set up disconnect handler
	manager.SetDisconnectHandler(func(deviceName, address string, err error) {
		fmt.Printf("⚠️  Device %s [%s] disconnected: %v\n", deviceName, address, err)
		fmt.Println("🔄 Will attempt to reconnect...")

		// Reset device state
		timeularDevice.Reset()
	})

	// Configure device for BLE manager
	deviceConfig := ble.DeviceConfig{
		Name:               timeularDevice.GetName(),
		ServiceUUID:        timeularDevice.GetServiceUUID(),
		CharacteristicUUID: timeularDevice.GetCharacteristicUUID(),
		NotificationHandler: func(deviceName string, data []byte) error {
			return timeularDevice.ProcessNotification(deviceName, data)
		},
	}

	// Start connecting to device
	fmt.Printf("🔍 Searching for Timeular tracker: %s\n", timeularDevice.GetName())
	fmt.Println("📱 Make sure your Timeular device is turned on and nearby!")
	if err := manager.ConnectDevices([]ble.DeviceConfig{deviceConfig}); err != nil {
		log.Fatalf("❌ Failed to start device connection: %v", err)
	}

	fmt.Println("✅ Connection process started")
	fmt.Printf("🎲 Device supports %d sides (1-%d)\n", timeular.GetSupportedSides(), timeular.GetSupportedSides())
	fmt.Printf("⚡ Polling interval: %v\n", 500*time.Millisecond)
	fmt.Println("📝 Rotate your Timeular device to different sides!")
	fmt.Println("🛑 Press Ctrl+C to stop")
	fmt.Println("")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")

	// Stop the device
	fmt.Println("🛑 Stopping Timeular device...")
	timeularDevice.Stop()

	// Clean shutdown
	fmt.Println("🧹 Cleaning up BLE connections...")
	if err := manager.Close(); err != nil {
		fmt.Printf("⚠️  Error during shutdown: %v\n", err)
	}

	fmt.Println("👋 Thanks for using the Timeular tracker!")
}
