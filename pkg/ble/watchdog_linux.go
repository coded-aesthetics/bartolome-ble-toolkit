//go:build linux

package ble

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"tinygo.org/x/bluetooth"
)

// watchConnection monitors a BLE device's Connected property via D-Bus.
// When the property changes to false, it calls onDisconnect with the
// bluetooth.Device. The returned cancel function stops the watcher.
func watchConnection(device *bluetooth.Device, addr bluetooth.Address, onDisconnect func(bluetooth.Device)) (cancel func()) {
	done := make(chan struct{})

	go func() {
		conn, err := dbus.SystemBus()
		if err != nil {
			fmt.Printf("⚠️  watchdog: cannot connect to D-Bus: %v\n", err)
			return
		}

		// Build the BlueZ device path from the MAC address.
		// Format: /org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF
		mac := addr.MAC.String()
		devPath := dbus.ObjectPath("/org/bluez/hci0/dev_" + strings.Replace(mac, ":", "_", -1))

		signal := make(chan *dbus.Signal, 1)
		conn.Signal(signal)
		matchOpts := []dbus.MatchOption{dbus.WithMatchInterface("org.freedesktop.DBus.Properties")}
		conn.AddMatchSignal(matchOpts...)

		defer func() {
			conn.RemoveSignal(signal)
			conn.RemoveMatchSignal(matchOpts...)
		}()

		for {
			select {
			case <-done:
				return
			case sig := <-signal:
				if sig == nil {
					return
				}
				if sig.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" {
					continue
				}
				if sig.Path != devPath {
					continue
				}
				if len(sig.Body) < 2 {
					continue
				}
				iface, ok := sig.Body[0].(string)
				if !ok || iface != "org.bluez.Device1" {
					continue
				}
				changes, ok := sig.Body[1].(map[string]dbus.Variant)
				if !ok {
					continue
				}
				if connected, ok := changes["Connected"]; ok {
					if val, ok := connected.Value().(bool); ok && !val {
						onDisconnect(*device)
						return
					}
				}
			}
		}
	}()

	return func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}
}
