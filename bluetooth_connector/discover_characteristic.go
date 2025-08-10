package bluetooth_connector

import (
	"tinygo.org/x/bluetooth"
)

func Discover_Characteristic(serviceUUID bluetooth.UUID, characteristicUUID bluetooth.UUID) (*bluetooth.DeviceCharacteristic, error) {
	devices_to_discover := []Device_To_Discover{
		{
			ServiceUUID:        serviceUUID,
			CharacteristicUUID: characteristicUUID,
		},
	}
	channel, err_channel, err := Discover_Multiple_Characteristics(devices_to_discover)

	if err != nil {
		return nil, err
	}

	select {
	case discovered_characteristic := <-channel:
		return discovered_characteristic.Characteristic, nil
	case err := <-err_channel:
		return nil, err
	}
}
