// Package timeular provides integration for Timeular tracker BLE devices.
// It handles connection, signal processing, and side detection for individual Timeular trackers.
package timeular

import (
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	// DefaultDeviceName is the default advertised name of Timeular tracker
	DefaultDeviceName = "Timeular Tracker"
	// DefaultPollInterval is the default interval for polling the device
	DefaultPollInterval = time.Second
)

var (
	// ServiceUUID is the custom Timeular service UUID
	ServiceUUID = bluetooth.NewUUID([16]byte{0xc7, 0xe7, 0x00, 0x10, 0xc8, 0x47, 0x11, 0xe6, 0x81, 0x75, 0x8c, 0x89, 0xa5, 0x5d, 0x40, 0x3c})
	// CharacteristicUUID is the Timeular characteristic UUID for data
	CharacteristicUUID = bluetooth.NewUUID([16]byte{0xc7, 0xe7, 0x00, 0x11, 0xc8, 0x47, 0x11, 0xe6, 0x81, 0x75, 0x8c, 0x89, 0xa5, 0x5d, 0x40, 0x3c})
)

// SideChangeHandler defines the function signature for handling side changes
type SideChangeHandler func(deviceName string, side byte) error

// DataHandler defines the function signature for handling raw data from the device
type DataHandler func(deviceName string, data []byte) error

// Device represents a single Timeular tracker device
type Device struct {
	name              string
	currentSide       byte
	lastSide          byte
	sideChangeHandler SideChangeHandler
	dataHandler       DataHandler
	stopChannel       chan bool
	running           bool
	pollInterval      time.Duration
	characteristic    *bluetooth.DeviceCharacteristic
}

// Config holds configuration options for a Timeular device
type Config struct {
	Name         string        // Custom name for this device instance
	PollInterval time.Duration // How often to poll for side changes
}

// NewDevice creates a new Timeular tracker device instance with default settings
func NewDevice() *Device {
	return &Device{
		name:         DefaultDeviceName,
		stopChannel:  make(chan bool, 1),
		pollInterval: DefaultPollInterval,
	}
}

// NewDeviceWithConfig creates a new Timeular tracker device with custom configuration
func NewDeviceWithConfig(config Config) *Device {
	device := NewDevice()

	if config.Name != "" {
		device.name = config.Name
	}

	if config.PollInterval > 0 {
		device.pollInterval = config.PollInterval
	}

	return device
}

// NewDeviceWithName creates a new Timeular device with a custom name
func NewDeviceWithName(name string) *Device {
	return NewDeviceWithConfig(Config{
		Name: name,
	})
}

// GetName returns the device name
func (d *Device) GetName() string {
	return d.name
}

// SetName updates the device name
func (d *Device) SetName(name string) {
	d.name = name
}

// GetServiceUUID returns the service UUID for the device
func (d *Device) GetServiceUUID() bluetooth.UUID {
	return ServiceUUID
}

// GetCharacteristicUUID returns the characteristic UUID for the device
func (d *Device) GetCharacteristicUUID() bluetooth.UUID {
	return CharacteristicUUID
}

// OnSideChange sets the handler function for side changes
func (d *Device) OnSideChange(handler SideChangeHandler) {
	d.sideChangeHandler = handler
}

// OnData sets the handler function for raw data (called before side processing)
func (d *Device) OnData(handler DataHandler) {
	d.dataHandler = handler
}

// SetPollInterval sets the interval for polling the device for side changes
func (d *Device) SetPollInterval(interval time.Duration) {
	d.pollInterval = interval
}

// GetCurrentSide returns the current side of the tracker
func (d *Device) GetCurrentSide() byte {
	return d.currentSide
}

// GetLastSide returns the previous side of the tracker
func (d *Device) GetLastSide() byte {
	return d.lastSide
}

// IsRunning returns whether the device is currently polling
func (d *Device) IsRunning() bool {
	return d.running
}

// ProcessNotification processes incoming BLE notifications from the tracker
// This is called by the BLE manager when data is received
// Note: Timeular devices typically use polling instead of notifications
func (d *Device) ProcessNotification(deviceName string, data []byte) error {
	// Call data handler if set
	if d.dataHandler != nil {
		if err := d.dataHandler(deviceName, data); err != nil {
			return fmt.Errorf("data handler error: %v", err)
		}
	}

	// Process the data for side detection
	if len(data) > 0 {
		return d.ProcessSideData(data)
	}

	// Start polling if not already running (for devices that don't send notifications)
	if !d.running {
		go d.startPolling()
	}

	return nil
}

// ProcessSideData processes raw data from the Timeular device to determine the current side
func (d *Device) ProcessSideData(data []byte) error {
	// Validate data
	if err := ValidateTimeularData(data); err != nil {
		return fmt.Errorf("invalid data: %v", err)
	}

	// Resolve side from data
	side, err := ResolveSide(data)
	if err != nil {
		return fmt.Errorf("failed to resolve side: %v", err)
	}

	// Update sides
	d.lastSide = d.currentSide
	d.currentSide = side

	// Call handler if side changed
	if d.currentSide != d.lastSide && d.sideChangeHandler != nil {
		return d.sideChangeHandler(d.name, d.currentSide)
	}

	return nil
}

// StartPolling manually starts the polling routine (usually not needed)
func (d *Device) StartPolling() {
	if !d.running {
		go d.startPolling()
	}
}

// startPolling starts the polling routine to read the device state
func (d *Device) startPolling() {
	d.running = true
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChannel:
			d.running = false
			return
		case <-ticker.C:
			// Poll the device for current state
			if err := d.pollDeviceState(); err != nil {
				// Log error but continue polling
				fmt.Printf("⚠️ Polling error for %s: %v\n", d.name, err)
			}
		}
	}
}

// pollDeviceState reads the current state from the device characteristic
func (d *Device) pollDeviceState() error {
	if d.characteristic == nil {
		return fmt.Errorf("characteristic not available")
	}

	// Read data from characteristic
	data := make([]byte, 12) // Timeular typically sends 12-byte data
	err := d.characteristic.Read(data)
	if err != nil {
		return fmt.Errorf("failed to read characteristic: %v", err)
	}

	// Process the data
	return d.ProcessSideData(data)
}

// SetCharacteristic sets the BLE characteristic for polling (used internally by BLE manager)
func (d *Device) SetCharacteristic(char *bluetooth.DeviceCharacteristic) {
	d.characteristic = char
}

// Stop stops the polling routine
func (d *Device) Stop() {
	if d.running {
		select {
		case d.stopChannel <- true:
		default:
		}
	}
}

// Reset resets the device state
func (d *Device) Reset() {
	d.Stop()
	d.currentSide = 0
	d.lastSide = 0
	d.characteristic = nil
}

// ResolveSide resolves the current side from Timeular device data
func ResolveSide(data []byte) (byte, error) {
	if err := ValidateTimeularData(data); err != nil {
		return 0, err
	}

	// Timeular side resolution logic
	// This is a simplified version - adjust based on actual Timeular protocol

	// Calculate side based on data pattern
	// Different algorithms can be implemented here based on your specific needs
	side := calculateSideFromData(data)

	// Ensure side is in valid range (1-8 for octagon)
	if side < 1 || side > 8 {
		return 0, fmt.Errorf("invalid side calculated: %d", side)
	}

	return side, nil
}

// calculateSideFromData implements the core algorithm for determining the side
func calculateSideFromData(data []byte) byte {
	// Simple implementation - replace with actual Timeular algorithm
	// This could involve analyzing accelerometer data, magnetometer data, etc.

	// For demonstration, use a simple hash-based approach
	sum := byte(0)
	for i, b := range data {
		sum += b * byte(i+1)
	}

	side := (sum % 8) + 1 // Sides 1-8
	return side
}

// FormatDataAsHex converts Timeular data to hex string format for debugging
func FormatDataAsHex(data []byte) string {
	return fmt.Sprintf("%x", data)
}

// ValidateTimeularData validates if the received data is a valid Timeular signal
func ValidateTimeularData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}

	if len(data) != 12 {
		return fmt.Errorf("invalid data length: expected 12 bytes, got %d", len(data))
	}

	// Check if data is not all zeros (which indicates invalid state)
	allZero := true
	for _, b := range data {
		if b != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		return fmt.Errorf("invalid data: all zeros")
	}

	return nil
}

// GetSupportedSides returns the number of sides supported by Timeular devices
func GetSupportedSides() int {
	return 8 // Standard Timeular tracker has 8 sides
}

// IsValidSide checks if a side number is valid for Timeular devices
func IsValidSide(side byte) bool {
	return side >= 1 && side <= 8
}

// Legacy function for backward compatibility
func Resolve_Side(data []byte) (byte, error) {
	return ResolveSide(data)
}
