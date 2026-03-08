//go:build !linux

package ble

import "tinygo.org/x/bluetooth"

// watchConnection is a no-op on non-Linux platforms where the adapter's
// SetConnectHandler already provides disconnect notifications.
func watchConnection(device *bluetooth.Device, addr bluetooth.Address, onDisconnect func(bluetooth.Device)) (cancel func()) {
	return func() {}
}
