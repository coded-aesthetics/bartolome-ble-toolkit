package bluetooth_connector

import (
	"fmt"
	"time"
)

func Connect_And_Reconnect_To_Devices(devices_to_discover []Device_To_Discover, listen_to_bluetooth_events func([]Discovered_Characteristic, chan bool)) {
	for {
		fmt.Println("🔄 Starting connection process...")

		stop_channel := make(chan bool, 1)
		disconnect_channel, err := Connect_To_Devices(stop_channel, devices_to_discover, listen_to_bluetooth_events)

		if err != nil {
			fmt.Printf("❌ Connection failed: %s\n", err.Error())
			fmt.Println("⏰ Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}

		fmt.Println("✅ All devices connected successfully")

		// Monitor for disconnections
		select {
		case disconnect_error := <-disconnect_channel:
			fmt.Printf("\n⚠️  Device disconnected: %s\n", disconnect_error.Error())

			// Signal all goroutines to stop
			select {
			case stop_channel <- true:
			default:
				// Channel might be full or closed
			}

			// Wait for cleanup
			time.Sleep(3 * time.Second)

			fmt.Println("🔄 Attempting to reconnect...")
		}

		// Brief delay before reconnection attempt
		time.Sleep(2 * time.Second)
	}
}

func Connect_To_Devices(stop_channel chan bool, devices_to_discover []Device_To_Discover, listen_to_bluetooth_events func([]Discovered_Characteristic, chan bool)) (chan error, error) {
	discovered_characteristics, disconnect_channel, err := Connect_To_Multiple_Characteristics(devices_to_discover)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to devices: %v", err)
	}

	fmt.Println("\n🎉 All devices connected successfully!")
	fmt.Printf("Connected devices: %d\n", len(discovered_characteristics))
	for _, char := range discovered_characteristics {
		fmt.Printf("  - %s [%s]\n", char.Name, char.Address.String())
	}
	fmt.Println("")

	// Start listening to events
	go listen_to_bluetooth_events(discovered_characteristics, stop_channel)

	return disconnect_channel, nil
}
