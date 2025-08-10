package timeular

import (
	"bluetooth_connector"
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

// Timeular Service UUID - c7e70010-c847-11e6-8175-8c89a55d403c
var TimeularServiceUUID = bluetooth.NewUUID([16]byte{0xc7, 0xe7, 0x00, 0x10, 0xc8, 0x47, 0x11, 0xe6, 0x81, 0x75, 0x8c, 0x89, 0xa5, 0x5d, 0x40, 0x3c})

// Timeular Characteristic - c7e70011-c847-11e6-8175-8c89a55d403c
var TimeularCharacteristicUUID = bluetooth.NewUUID([16]byte{0xc7, 0xe7, 0x00, 0x11, 0xc8, 0x47, 0x11, 0xe6, 0x81, 0x75, 0x8c, 0x89, 0xa5, 0x5d, 0x40, 0x3c})

var Timeular_Device_Name = "Timeular Tracker"

var Timeular_Device = bluetooth_connector.Device_To_Discover{
	Name:                              Timeular_Device_Name,
	ServiceUUID:                       TimeularServiceUUID,
	CharacteristicUUID:                TimeularCharacteristicUUID,
	Establish_Characteristic_Listener: Establish_Timeular_Characteristic_Listener,
}

var Timeular_Device_Name_2 = "Timeular Tracker 2"

var Timeular_Device_2 = bluetooth_connector.Device_To_Discover{
	Name:                              Timeular_Device_Name_2,
	ServiceUUID:                       TimeularServiceUUID,
	CharacteristicUUID:                TimeularCharacteristicUUID,
	Establish_Characteristic_Listener: Establish_Timeular_Characteristic_Listener,
}

func Establish_Timeular_Characteristic_Listener(characteristic bluetooth.DeviceCharacteristic) (chan []byte, func(), error) {
	channel := make(chan []byte)
	stop_channel := make(chan bool)
	go read_timeular_side_info(characteristic, stop_channel, channel)

	return channel, func() {
		stop_channel <- true
		close(stop_channel)
	}, nil
}

func read_timeular_side_info(characteristic bluetooth.DeviceCharacteristic, stop_channel chan bool, channel chan []byte) {
	old_side := ""

	tick := time.NewTicker(time.Second)

	for {
		select {
		case <-stop_channel:
			tick.Stop()
			return
		case <-tick.C:
			payload := make([]byte, 12)
			characteristic.Read(payload)

			new_side := fmt.Sprintf("%x", payload)

			if old_side != new_side {
				if len(payload) == 12 {
					channel <- payload
				}
			}
			old_side = new_side
		}
	}
}
