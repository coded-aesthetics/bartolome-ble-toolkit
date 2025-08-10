// Package columbus provides integration for the Columbus Video Pen BLE device.
// It handles connection, signal processing, and country resolution for the pen.
package columbus

import (
	"fmt"

	"tinygo.org/x/bluetooth"
)

const (
	// DeviceName is the advertised name of the Columbus Video Pen
	DeviceName = "COLUMBUS Video Pen"
)

var (
	// ServiceUUID is the Nordic UART service UUID used by Columbus Video Pen
	ServiceUUID = bluetooth.ServiceUUIDNordicUART
	// CharacteristicUUID is the UART TX characteristic UUID for notifications
	CharacteristicUUID = bluetooth.CharacteristicUUIDUARTTX
)

// SignalHandler defines the function signature for handling pen signals
type SignalHandler func(signal []byte) error

// Device represents a Columbus Video Pen device
type Device struct {
	name           string
	signalHandler  SignalHandler
	connected      bool
	lastSignal     []byte
	validationFunc func([]byte) bool
}

// NewDevice creates a new Columbus Video Pen device instance
func NewDevice() *Device {
	return &Device{
		name:           DeviceName,
		validationFunc: DefaultSignalValidator,
	}
}

// GetName returns the device name
func (d *Device) GetName() string {
	return d.name
}

// GetServiceUUID returns the service UUID for the device
func (d *Device) GetServiceUUID() bluetooth.UUID {
	return ServiceUUID
}

// GetCharacteristicUUID returns the characteristic UUID for the device
func (d *Device) GetCharacteristicUUID() bluetooth.UUID {
	return CharacteristicUUID
}

// OnSignal sets the handler function for incoming pen signals
func (d *Device) OnSignal(handler SignalHandler) {
	d.signalHandler = handler
}

// SetSignalValidator sets a custom validation function for signals
// The validator should return true if the signal is valid
func (d *Device) SetSignalValidator(validator func([]byte) bool) {
	d.validationFunc = validator
}

// ProcessNotification processes incoming BLE notifications from the pen
// This is called by the BLE manager when data is received
func (d *Device) ProcessNotification(deviceName string, data []byte) error {
	// Validate the signal
	if !d.isValidSignal(data) {
		return fmt.Errorf("invalid signal received: %x", data)
	}

	// Store the last signal
	d.lastSignal = make([]byte, len(data))
	copy(d.lastSignal, data)

	// Call the signal handler if set
	if d.signalHandler != nil {
		return d.signalHandler(data)
	}

	return nil
}

// GetLastSignal returns the last valid signal received from the pen
func (d *Device) GetLastSignal() []byte {
	if d.lastSignal == nil {
		return nil
	}
	result := make([]byte, len(d.lastSignal))
	copy(result, d.lastSignal)
	return result
}

// isValidSignal checks if the received signal is valid
func (d *Device) isValidSignal(signal []byte) bool {
	// Check for empty signals (device disconnecting)
	if len(signal) == 0 {
		return false
	}

	// Use custom validator if set
	if d.validationFunc != nil {
		return d.validationFunc(signal)
	}

	return true
}

// DefaultSignalValidator is the default validation function for Columbus pen signals
func DefaultSignalValidator(signal []byte) bool {
	// Basic validation: signal should not be empty and should have reasonable length
	if len(signal) == 0 {
		return false
	}

	// Columbus pen typically sends signals with specific patterns
	// You can customize this based on your specific requirements
	return len(signal) >= 4 // Minimum expected signal length
}

// FormatSignalAsHex converts a signal to hex string format
func FormatSignalAsHex(signal []byte) string {
	return fmt.Sprintf("%x", signal)
}

// SignalToCountryHex extracts the country hex part from a Columbus pen signal
// This assumes the country code is at bytes 5-6 (positions 10-13 in hex string)
func SignalToCountryHex(signal []byte) (string, error) {
	hexStr := FormatSignalAsHex(signal)

	// Check if hex string is long enough for country extraction
	if len(hexStr) < 14 {
		return "", fmt.Errorf("signal too short for country extraction: %s (length: %d)", hexStr, len(hexStr))
	}

	// Extract country hex (positions 10-13 in hex string)
	countryHex := hexStr[10:14]
	return countryHex, nil
}
