package bluetooth_connector

import (
	"fmt"
	"utils"

	"tinygo.org/x/bluetooth"
)

func Connect_To_Multiple_Characteristics(devices_to_discover []Device_To_Discover) ([]Discovered_Characteristic, chan error, error) {
	channel, err_channel, err := Discover_Multiple_Characteristics(devices_to_discover)

	if err != nil {
		return nil, nil, err
	}

	discovered_characteristics := make([]Discovered_Characteristic, 0)
	for {
		select {
		case discovered_characteristic := <-channel:
			discovered_characteristics = append(discovered_characteristics, *discovered_characteristic)
			if len(discovered_characteristics) == len(devices_to_discover) {
				error_channel := make(chan error)

				chars_copy := make([]Discovered_Characteristic, len(discovered_characteristics))
				copy(chars_copy, discovered_characteristics)

				go Setup_Disconnect_Listener(error_channel, chars_copy)

				return discovered_characteristics, error_channel, nil
			}
		case err := <-err_channel:
			return nil, nil, err
		}

	}
}

func Setup_Disconnect_Listener(error_channel chan error, discovered_characteristics []Discovered_Characteristic) {
	var adapter = bluetooth.DefaultAdapter

	adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
		found_devices := utils.Filter_Array(discovered_characteristics, func(discovered_characteristic Discovered_Characteristic) bool {
			return discovered_characteristic.Address.String() == device.Address.String()
		})
		if len(found_devices) > 0 && !connected {
			error_channel <- fmt.Errorf("%s [%s] disconnected", found_devices[0].Name, device.Address.String())
			disconnect_all(discovered_characteristics)
		}
	})

}

func disconnect_all(discovered_characteristics []Discovered_Characteristic) {
	for _, device := range discovered_characteristics {
		if device.Disable_Characteristic_Listener != nil {
			device.Disable_Characteristic_Listener()
		}
		if device.Device != nil {
			device.Device.Disconnect()
		}
		// Don't close the channel here as it might still be in use
		// The channel will be garbage collected when no longer referenced
	}
}
