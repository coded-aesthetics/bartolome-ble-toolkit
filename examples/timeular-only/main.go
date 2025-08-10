package main

import (
	"fmt"
	"log"
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
		Name:         "My Timeular Tracker",
		PollInterval: 500 * time.Millisecond, // Poll every 500ms for faster response
	})

	// Create a BLE manager
	manager := ble.NewManager()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Track statistics
	var (
		totalSideChanges int
		lastChangeTime   time.Time
		startTime        = time.Now()
	)

	// Set up side change handler
	timeularDevice.OnSideChange(func(deviceName string, side byte) error {
		now := time.Now()
		totalSideChanges++

		if !lastChangeTime.IsZero() {
			timeSinceLastChange := now.Sub(lastChangeTime)
			fmt.Printf("🎲 %s side changed: %d (after %.1fs)\n",
				deviceName, side, timeSinceLastChange.Seconds())
		} else {
			fmt.Printf("🎲 %s initial side: %d\n", deviceName, side)
		}

		lastChangeTime = now

		// Validate the side
		if !timeular.IsValidSide(side) {
			fmt.Printf("⚠️  Warning: Invalid side detected: %d\n", side)
			return fmt.Errorf("invalid side: %d", side)
		}

		// Example: trigger different actions based on side
		switch side {
		case 1:
			fmt.Println("   📝 Action: Work mode activated")
		case 2:
			fmt.Println("   ☕ Action: Break time")
		case 3:
			fmt.Println("   📞 Action: Meeting mode")
		case 4:
			fmt.Println("   🎯 Action: Focus time")
		case 5:
			fmt.Println("   📚 Action: Learning mode")
		case 6:
			fmt.Println("   🏃 Action: Exercise time")
		case 7:
			fmt.Println("   🍽️ Action: Meal time")
		case 8:
			fmt.Println("   😴 Action: Rest mode")
		default:
			fmt.Printf("   ❓ Action: Unknown side %d\n", side)
		}

		fmt.Printf("   📊 Total changes: %d, Session time: %.1fm\n",
			totalSideChanges, time.Since(startTime).Minutes())
		fmt.Println("")

		return nil
	})

	// Optional: Handle raw data for debugging
	timeularDevice.OnData(func(deviceName string, data []byte) error {
		if len(data) > 0 {
			fmt.Printf("📊 Raw data from %s: [%x] (length: %d)\n", deviceName, data, len(data))

			// Validate the data
			if err := timeular.ValidateTimeularData(data); err != nil {
				fmt.Printf("⚠️  Data validation failed: %v\n", err)
				return nil // Don't return error to avoid disconnection
			}
		}
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

	// Display periodic status updates
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sigChan:
				return
			case <-ticker.C:
				if manager.IsConnected(timeularDevice.GetName()) {
					currentSide := timeularDevice.GetCurrentSide()
					lastSide := timeularDevice.GetLastSide()
					sessionTime := time.Since(startTime)

					fmt.Printf("📊 Status Update:\n")
					fmt.Printf("   Device: %s (connected)\n", timeularDevice.GetName())
					fmt.Printf("   Current side: %d, Previous side: %d\n", currentSide, lastSide)
					fmt.Printf("   Total changes: %d\n", totalSideChanges)
					fmt.Printf("   Session time: %.1f minutes\n", sessionTime.Minutes())
					fmt.Printf("   Is polling: %v\n", timeularDevice.IsRunning())
					fmt.Println("")
				} else {
					fmt.Printf("📊 Status: Device %s not connected\n", timeularDevice.GetName())
				}
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
	fmt.Printf("   Total side changes: %d\n", totalSideChanges)
	if sessionTime.Minutes() > 0 {
		changesPerMinute := float64(totalSideChanges) / sessionTime.Minutes()
		fmt.Printf("   Average changes per minute: %.1f\n", changesPerMinute)
	}
	fmt.Printf("   Final side: %d\n", timeularDevice.GetCurrentSide())

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
