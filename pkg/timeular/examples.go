// Package timeular provides examples and configuration patterns for Timeular tracker devices.
// This file demonstrates various ways to configure and use Timeular devices.
package timeular

import (
	"fmt"
	"time"

	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/ble"
)

// ExampleUsage demonstrates basic usage patterns for Timeular devices
func ExampleUsage() {
	// Example 1: Create a device with default settings
	device1 := NewDevice()
	fmt.Printf("Device 1: %s\n", device1.GetName()) // "Timeular Tracker"

	// Example 2: Create a device with custom name
	device2 := NewDeviceWithName("My Work Tracker")
	fmt.Printf("Device 2: %s\n", device2.GetName()) // "My Work Tracker"

	// Example 3: Create a device with full configuration
	device3 := NewDeviceWithConfig(Config{
		Name:         "Home Tracker",
		PollInterval: 250 * time.Millisecond, // Fast polling
	})
	fmt.Printf("Device 3: %s, Poll interval: %v\n", device3.GetName(), 250*time.Millisecond)

	// Example 4: Multiple devices for different purposes
	workTracker := NewDeviceWithName("Work Timeular")
	homeTracker := NewDeviceWithName("Home Timeular")

	// Configure different polling rates
	workTracker.SetPollInterval(500 * time.Millisecond)  // Medium speed for work
	homeTracker.SetPollInterval(1000 * time.Millisecond) // Slower for home use
}

// ExampleHandlers demonstrates how to set up event handlers
func ExampleHandlers() {
	device := NewDeviceWithName("Example Tracker")

	// Basic side change handler
	device.OnSideChange(func(deviceName string, side byte) error {
		fmt.Printf("Device %s changed to side %d\n", deviceName, side)
		return nil
	})

	// Raw data handler for debugging
	device.OnData(func(deviceName string, data []byte) error {
		fmt.Printf("Raw data from %s: %x\n", deviceName, data)
		return nil
	})

	// Advanced side change handler with validation
	device.OnSideChange(func(deviceName string, side byte) error {
		if !IsValidSide(side) {
			return fmt.Errorf("invalid side: %d", side)
		}

		// Map sides to activities
		activities := map[byte]string{
			1: "Work",
			2: "Break",
			3: "Meeting",
			4: "Focus Time",
			5: "Learning",
			6: "Exercise",
			7: "Personal",
			8: "Rest",
		}

		if activity, exists := activities[side]; exists {
			fmt.Printf("%s: Starting %s\n", deviceName, activity)
		}

		return nil
	})
}

// ExampleBLEConfiguration shows how to configure devices for the BLE manager
func ExampleBLEConfiguration() []ble.DeviceConfig {
	// Create multiple Timeular devices
	tracker1 := NewDeviceWithName("Office Tracker")
	tracker2 := NewDeviceWithName("Home Tracker")

	// Configure both devices
	configs := []ble.DeviceConfig{
		{
			Name:               tracker1.GetName(),
			ServiceUUID:        tracker1.GetServiceUUID(),
			CharacteristicUUID: tracker1.GetCharacteristicUUID(),
			NotificationHandler: func(deviceName string, data []byte) error {
				return tracker1.ProcessNotification(deviceName, data)
			},
		},
		{
			Name:               tracker2.GetName(),
			ServiceUUID:        tracker2.GetServiceUUID(),
			CharacteristicUUID: tracker2.GetCharacteristicUUID(),
			NotificationHandler: func(deviceName string, data []byte) error {
				return tracker2.ProcessNotification(deviceName, data)
			},
		},
	}

	return configs
}

// ExampleActivityTracking demonstrates a complete activity tracking setup
func ExampleActivityTracking() {
	// Activity mapping
	type Activity struct {
		Name     string
		Color    string
		Billable bool
		Category string
	}

	activities := map[byte]Activity{
		1: {"Development", "blue", true, "Work"},
		2: {"Code Review", "green", true, "Work"},
		3: {"Meetings", "purple", true, "Work"},
		4: {"Planning", "orange", true, "Work"},
		5: {"Learning", "yellow", false, "Development"},
		6: {"Break", "gray", false, "Personal"},
		7: {"Admin", "red", false, "Work"},
		8: {"Idle", "black", false, "Personal"},
	}

	// Create tracker
	tracker := NewDeviceWithConfig(Config{
		Name:         "Activity Tracker",
		PollInterval: 500 * time.Millisecond,
	})

	// Track current activity
	var currentActivity *Activity
	var activityStartTime time.Time

	tracker.OnSideChange(func(deviceName string, side byte) error {
		now := time.Now()

		// Log previous activity duration
		if currentActivity != nil && !activityStartTime.IsZero() {
			duration := now.Sub(activityStartTime)
			fmt.Printf("Completed: %s for %.1f minutes\n",
				currentActivity.Name, duration.Minutes())
		}

		// Start new activity
		if activity, exists := activities[side]; exists {
			currentActivity = &activity
			activityStartTime = now

			fmt.Printf("Started: %s (%s) - Billable: %v\n",
				activity.Name, activity.Category, activity.Billable)
		} else {
			fmt.Printf("Unknown activity for side %d\n", side)
			currentActivity = nil
		}

		return nil
	})
}

// ExampleMultiDeviceSetup shows how to handle multiple Timeular devices
func ExampleMultiDeviceSetup() {
	// Create devices for different contexts
	devices := []*Device{
		NewDeviceWithName("Work Tracker"),
		NewDeviceWithName("Personal Tracker"),
		NewDeviceWithName("Gym Tracker"),
	}

	// Set up different configurations for each
	for i, device := range devices {
		// Different polling intervals
		pollInterval := time.Duration(500+i*250) * time.Millisecond
		device.SetPollInterval(pollInterval)

		// Device-specific handlers
		deviceIndex := i // Capture for closure
		device.OnSideChange(func(deviceName string, side byte) error {
			fmt.Printf("Device %d (%s): Side %d\n", deviceIndex+1, deviceName, side)

			// Different logic per device
			switch deviceIndex {
			case 0: // Work tracker
				return handleWorkActivity(side)
			case 1: // Personal tracker
				return handlePersonalActivity(side)
			case 2: // Gym tracker
				return handleGymActivity(side)
			}
			return nil
		})
	}
}

// Helper functions for different activity types
func handleWorkActivity(side byte) error {
	workActivities := map[byte]string{
		1: "Coding", 2: "Code Review", 3: "Meetings", 4: "Planning",
		5: "Documentation", 6: "Testing", 7: "Admin", 8: "Break",
	}

	if activity, exists := workActivities[side]; exists {
		fmt.Printf("Work: %s\n", activity)
	}
	return nil
}

func handlePersonalActivity(side byte) error {
	personalActivities := map[byte]string{
		1: "Reading", 2: "Exercise", 3: "Cooking", 4: "Cleaning",
		5: "Hobbies", 6: "Social", 7: "Entertainment", 8: "Rest",
	}

	if activity, exists := personalActivities[side]; exists {
		fmt.Printf("Personal: %s\n", activity)
	}
	return nil
}

func handleGymActivity(side byte) error {
	gymActivities := map[byte]string{
		1: "Cardio", 2: "Strength", 3: "Stretching", 4: "Core",
		5: "Arms", 6: "Legs", 7: "Back", 8: "Rest",
	}

	if activity, exists := gymActivities[side]; exists {
		fmt.Printf("Gym: %s\n", activity)
	}
	return nil
}

// ExampleErrorHandling demonstrates proper error handling patterns
func ExampleErrorHandling() {
	device := NewDeviceWithName("Error Example")

	// Handler with comprehensive error handling
	device.OnSideChange(func(deviceName string, side byte) error {
		// Validate side
		if !IsValidSide(side) {
			fmt.Printf("Error: Invalid side %d for device %s\n", side, deviceName)
			return fmt.Errorf("invalid side: %d", side)
		}

		// Check for rapid changes (might indicate device issues)
		if device.GetLastSide() != 0 && device.GetCurrentSide() != device.GetLastSide() {
			// Side changed very quickly, might be noise
			fmt.Printf("Warning: Rapid side change detected on %s\n", deviceName)
		}

		// Successful processing
		fmt.Printf("Valid side change: %s -> side %d\n", deviceName, side)
		return nil
	})

	// Data handler with validation
	device.OnData(func(deviceName string, data []byte) error {
		if err := ValidateTimeularData(data); err != nil {
			fmt.Printf("Data validation failed for %s: %v\n", deviceName, err)
			return nil // Don't return error to avoid disconnection
		}

		// Process valid data
		fmt.Printf("Valid data received from %s: %s\n", deviceName, FormatDataAsHex(data))
		return nil
	})
}

// ExampleCleanup demonstrates proper cleanup patterns
func ExampleCleanup() {
	device := NewDeviceWithName("Cleanup Example")

	// Use the device...

	// Cleanup when done
	device.Stop()  // Stop polling
	device.Reset() // Reset state

	fmt.Printf("Device %s cleaned up\n", device.GetName())
}
