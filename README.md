# Bartolome BLE Toolkit

A modular Go library for working with Bluetooth Low Energy devices, specifically designed for Columbus Video Pens and Timeular trackers. Built with robust connection management, automatic reconnection, and clean, reusable APIs.

## 🚀 Features

- **Modular Design**: Independent packages for different device types
- **Robust Connection Management**: Automatic reconnection and error handling
- **macOS Compatible**: Tested and optimized for macOS Bluetooth stack
- **Country Resolution**: Built-in country detection from Columbus pen signals
- **Real-time Processing**: Handle multiple devices simultaneously
- **Clean APIs**: Easy to integrate and extend

## 📦 Packages

### Core BLE Package (`pkg/ble`)
High-level BLE device management with automatic connection and reconnection.

### Columbus Package (`pkg/columbus`)
Integration for Columbus Video Pen devices with country detection.

### Timeular Package (`pkg/timeular`)
Support for Timeular tracker devices with side detection.

### Countries Package (`pkg/countries`)
Country resolution from hex codes and signal processing.

## 🛠️ Installation

```bash
go get github.com/coded-aesthetics/bartolome-ble-toolkit
```

## 📱 Quick Start

### Columbus Video Pen Only

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/ble"
    "github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/columbus"
    "github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/countries"
)

func main() {
    // Create devices
    columbusDevice := columbus.NewDevice()
    manager := ble.NewManager()

    // Set up signal handler
    columbusDevice.OnSignal(func(signal []byte) error {
        countryHex, _ := columbus.SignalToCountryHex(signal)
        country, _ := countries.ResolveFromHex(countryHex)
        fmt.Printf("Country detected: %s\n", country.Name)
        return nil
    })

    // Configure and connect
    deviceConfig := ble.DeviceConfig{
        Name:               columbusDevice.GetName(),
        ServiceUUID:        columbusDevice.GetServiceUUID(), 
        CharacteristicUUID: columbusDevice.GetCharacteristicUUID(),
        NotificationHandler: columbusDevice.ProcessNotification,
    }

    if err := manager.ConnectDevices([]ble.DeviceConfig{deviceConfig}); err != nil {
        log.Fatal(err)
    }

    // Keep running...
    select {}
}
```

### Multiple Devices

```go
// Create all devices
columbusDevice := columbus.NewDevice()
timeularDevice1 := timeular.NewDeviceWithName("Timeular Tracker 1")
timeularDevice2 := timeular.NewDeviceWithName("Timeular Tracker 2")

// Set up handlers
columbusDevice.OnSignal(func(signal []byte) error {
    // Handle Columbus pen signals
    return nil
})

timeularDevice1.OnSideChange(func(deviceName string, side byte) error {
    fmt.Printf("Timeular 1 side: %d\n", side)
    return nil
})

// Configure all devices
deviceConfigs := []ble.DeviceConfig{
    {
        Name: columbusDevice.GetName(),
        ServiceUUID: columbusDevice.GetServiceUUID(),
        CharacteristicUUID: columbusDevice.GetCharacteristicUUID(),
        NotificationHandler: columbusDevice.ProcessNotification,
    },
    {
        Name: timeularDevice1.GetName(),
        ServiceUUID: timeularDevice1.GetServiceUUID(),
        CharacteristicUUID: timeularDevice1.GetCharacteristicUUID(),
        NotificationHandler: timeularDevice1.ProcessNotification,
    },
    // Add more devices...
}

manager.ConnectDevices(deviceConfigs)
```

## 🎯 Examples

The `examples/` directory contains complete working examples:

- **`columbus-only/`**: Simple Columbus Video Pen integration
- **`timeular-only/`**: Single Timeular tracker example
- **`full-setup/`**: Complete setup with all supported devices

Run examples:
```bash
# Columbus Video Pen only
cd examples/columbus-only
go run main.go

# Single Timeular tracker
cd examples/timeular-only
go run main.go

# Full setup with all devices
cd examples/full-setup
go run main.go
```

## 🔧 API Reference

### BLE Manager

```go
type Manager struct {
    // Core BLE device management
}

func NewManager() *Manager
func (m *Manager) ConnectDevices(configs []DeviceConfig) error
func (m *Manager) SetDisconnectHandler(handler func(string, string, error))
func (m *Manager) GetConnectedDevices() map[string]*ConnectedDevice
func (m *Manager) IsConnected(deviceName string) bool
func (m *Manager) Close() error
```

### Columbus Device

```go
type Device struct {
    // Columbus Video Pen device
}

func NewDevice() *Device
func (d *Device) OnSignal(handler SignalHandler)
func (d *Device) GetLastSignal() []byte
func (d *Device) SetSignalValidator(validator func([]byte) bool)

// Utility functions
func SignalToCountryHex(signal []byte) (string, error)
func FormatSignalAsHex(signal []byte) string
```

### Country Resolution

```go
type Resolver struct {
    // Country resolution engine
}

func NewResolver() *Resolver
func (r *Resolver) ResolveFromSignal(signal []byte) (*Country, error)
func (r *Resolver) ResolveFromHex(hex string) (*Country, error)
func (r *Resolver) ResolveFromCountryCode(code int) (*Country, error)

// Convenience functions
func ResolveFromSignal(signal []byte) (*Country, error)
func ResolveFromHex(hex string) (*Country, error)
```

### Timeular Device

```go
type Device struct {
    // Timeular tracker device
}

func NewDevice() *Device
func NewDeviceWithName(name string) *Device
func NewDeviceWithConfig(config Config) *Device
func (d *Device) OnSideChange(handler SideChangeHandler)
func (d *Device) OnData(handler DataHandler)
func (d *Device) GetCurrentSide() byte
func (d *Device) GetLastSide() byte
func (d *Device) SetPollInterval(interval time.Duration)
func (d *Device) IsRunning() bool
func (d *Device) Stop()
func (d *Device) Reset()

// Utility functions  
func ResolveSide(data []byte) (byte, error)
func ValidateTimeularData(data []byte) error
```

## 🔍 Device Support

### Columbus Video Pen
- **Service**: Nordic UART (`6e400001-b5a3-f393-e0a9-e50e24dcca9e`)
- **Characteristic**: UART TX (`6e400003-b5a3-f393-e0a9-e50e24dcca9e`)
- **Features**: Country detection, signal validation, real-time notifications

### Timeular Tracker
- **Service**: Custom Timeular service (`c7e70010-c847-11e6-8175-8c89a55d403c`)
- **Characteristic**: Custom characteristic (`c7e70011-c847-11e6-8175-8c89a55d403c`)
- **Features**: Side detection, polling-based updates, modular single-device design
- **Supported Sides**: 1-8 (standard octagon)
- **Usage**: Create multiple instances for multiple devices

## 🍎 macOS Compatibility

This library is specifically tested and optimized for macOS:

- **Bluetooth Permissions**: Handles macOS permission requirements
- **Connection Management**: Works around macOS BLE stack limitations
- **Scanning Behavior**: Stops scanning before connecting (required on macOS)
- **Reconnection Logic**: Robust reconnection handling for macOS

### macOS Setup

1. Grant Bluetooth permissions to Terminal or your IDE:
   - System Settings → Privacy & Security → Bluetooth
   - Add Terminal/your IDE to allowed apps

2. Ensure Bluetooth is enabled in System Settings

## 🛠️ Development

### Building

```bash
go build ./...
```

### Testing

```bash
go test ./...
```

### Adding New Devices

1. Create a new package in `pkg/yourdevice/`
2. Implement the device interface:
   ```go
   func (d *Device) GetName() string
   func (d *Device) GetServiceUUID() bluetooth.UUID
   func (d *Device) GetCharacteristicUUID() bluetooth.UUID
   func (d *Device) ProcessNotification(deviceName string, data []byte) error
   ```
3. Add configuration to examples

## 📁 Project Structure

```
bartolome-ble-toolkit/
├── README.md
├── go.mod
├── pkg/
│   ├── ble/           # Core BLE management
│   ├── columbus/      # Columbus Video Pen
│   ├── timeular/      # Timeular trackers
│   └── countries/     # Country resolution
├── examples/
│   ├── columbus-only/    # Simple Columbus example
│   ├── timeular-only/    # Single Timeular example
│   ├── full-setup/       # Complete multi-device example
│   └── working-columbus/ # Reliable working example
└── docs/                 # Additional documentation
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [TinyGo Bluetooth](https://github.com/tinygo-org/bluetooth) for excellent BLE support
- Columbus and Timeular for creating innovative BLE devices
- The Go community for fantastic tooling and libraries

## 🐛 Troubleshooting

### Common Issues

**"Could not enable BLE adapter"**
- Check Bluetooth permissions in macOS System Settings
- Ensure Bluetooth is enabled
- Try running with `sudo` (not recommended for production)

**"Connection timeout"** 
- Ensure devices are powered on and nearby
- Check that devices aren't connected to other applications
- Verify device names match exactly

**"Service/Characteristic not found"**
- Confirm device UUIDs are correct
- Some devices may not advertise all services
- Try connecting with a generic BLE scanner first

### Debug Mode

Enable debug logging:
```go
import "log"

// Add detailed logging in your handlers
columbus.OnSignal(func(signal []byte) error {
    log.Printf("Raw signal: %x", signal)
    // ... rest of handler
})
```

## 📞 Support

- Create an issue on GitHub for bugs or feature requests
- Check existing issues for known problems and solutions
- See examples/ directory for working code references

---

Built with ❤️ for the BLE community