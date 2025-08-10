package columbus

import (
	"bluetooth_connector"
	"fmt"

	"tinygo.org/x/bluetooth"
)

var ColumbusPenServiceUUID = bluetooth.ServiceUUIDNordicUART
var ColumbusPenCharacteristicUUID = bluetooth.CharacteristicUUIDUARTTX

var Columbus_Device_Name = "COLUMBUS Video Pen"

var Columbus_Device = bluetooth_connector.Device_To_Discover{
	Name:                              Columbus_Device_Name,
	ServiceUUID:                       ColumbusPenServiceUUID,
	CharacteristicUUID:                ColumbusPenCharacteristicUUID,
	Establish_Characteristic_Listener: Establish_Pen_Characteristic_Listener,
}

func Establish_Pen_Characteristic_Listener(characteristic bluetooth.DeviceCharacteristic) (chan []byte, func(), error) {
	channel := make(chan []byte, 10) // Buffered channel to prevent blocking

	// Enable notifications to receive incoming data.
	err := characteristic.EnableNotifications(func(value []byte) {
		// Safety check: only send to channel if it's not closed
		select {
		case channel <- value:
			// Successfully sent
		default:
			// Channel is full or closed, drop the data
			fmt.Printf("⚠️  Dropped notification data - channel unavailable\n")
		}
	})

	if err != nil {
		println("Failed to enable notifications:", err.Error())
		return nil, nil, err
	}

	return channel, func() {
		characteristic.EnableNotifications(nil)
		// Don't close the channel here as it might still be in use
	}, nil
}
