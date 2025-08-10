# Examples Guide - Bartolome BLE Toolkit

This guide explains how to run the example applications in the Bartolome BLE Toolkit.

## 🚨 Important: Running Examples

Due to Go workspace configuration, examples must be run with the `GOWORK=off` environment variable to avoid module conflicts.

## 📋 Prerequisites

1. **macOS with Bluetooth enabled**
2. **Go 1.21 or later**
3. **Bluetooth permissions granted to Terminal**
   - Go to System Settings → Privacy & Security → Bluetooth
   - Add Terminal to allowed applications
4. **Your BLE devices powered on and nearby**

## 🖊️ Columbus Video Pen Example

### Simple Working Example (Recommended)
```bash
cd examples/working-columbus
go mod tidy
GOWORK=off go run main.go
```

This example provides:
- ✅ Reliable connection to Columbus Video Pen
- ✅ Signal reception and country detection
- ✅ Simple, straightforward code
- ✅ Statistics and status monitoring

### Modular Example
```bash
cd examples/columbus-only
go mod tidy
GOWORK=off go run main.go
```

This example demonstrates:
- ✅ Using the modular BLE manager
- ✅ Columbus and Countries packages
- ✅ Automatic reconnection handling

## 🎲 Timeular Tracker Example

```bash
cd examples/timeular-only
go mod tidy
GOWORK=off go run main.go
```

Features:
- ✅ Single Timeular device management
- ✅ Side detection and validation
- ✅ Configurable polling intervals
- ✅ Statistics tracking

## 🚀 Full Setup Example

```bash
cd examples/full-setup
go mod tidy
GOWORK=off go run main.go
```

Demonstrates:
- ✅ Multiple device management
- ✅ Columbus + 2 Timeular devices
- ✅ Combined signal processing
- ✅ Action triggering simulation

## 🔍 Troubleshooting

### Module Import Errors
If you see module import errors, ensure you're using `GOWORK=off`:
```bash
GOWORK=off go run main.go
```

### Bluetooth Permission Issues
- Check System Settings → Privacy & Security → Bluetooth
- Ensure Terminal has Bluetooth access
- Try running with `sudo` if needed (not recommended for production)

### Device Not Found
- Ensure your device is powered on
- Check device is not connected to other applications
- Verify device is within range (< 10 meters)
- Try turning Bluetooth off and on

### Connection Timeouts
- Wait for device to fully disconnect from other apps
- Restart Bluetooth service: `sudo pkill bluetoothd`
- Try restarting the application

## 📊 Example Outputs

### Working Columbus Example
```
🖊️  Columbus Video Pen - Working Example
========================================
🚀 Initializing connection...
🔌 Enabling BLE adapter...
✅ BLE adapter enabled
🔍 Scanning for COLUMBUS Video Pen...
📱 Found COLUMBUS Video Pen [address] RSSI: -54
🔗 Connecting to COLUMBUS Video Pen...
✅ Device connected
🔍 Discovering services...
✅ Found Nordic UART service
🔍 Discovering characteristics...
✅ Found UART TX characteristic
🔔 Setting up notifications...
✅ Notifications enabled
🎉 Columbus Video Pen connected and ready!
📝 Tap your Columbus Video Pen on different locations!
🛑 Press Ctrl+C to stop

🖊️  Signal #1 received: [0ea00000003b1d00] (length: 8)
🌍 Country: Unknown Country (3b1d) (XX)
🗺️  Region: Unknown
🔢 Country Code: 3b1d
🎯 ACTION: Triggering request for Unknown Country (3b1d)
📊 Session stats: 1 signals in 0.2 minutes
```

### Timeular Example
```
🎲 Timeular Tracker Example
===========================
🔍 Searching for Timeular tracker: My Timeular Tracker
📱 Make sure your Timeular device is turned on!
✅ Connection process started
🎲 Device supports 8 sides (1-8)
⚡ Polling interval: 500ms
📝 Rotate your Timeular device to different sides!

🎲 My Timeular Tracker side changed: 3 (after 2.3s)
   📞 Action: Meeting mode
   📊 Total changes: 1, Session time: 0.1m
```

## 🛠️ Development Tips

### Local Development
Examples use replace directives to work with local code:
```go
replace github.com/coded-aesthetics/bartolome-ble-toolkit => ../..
```

### Testing Module Imports
Test the module directly:
```bash
cd test-external
GOWORK=off go run main.go
```

### Creating Your Own Example
1. Create new directory outside the main module
2. Initialize with `go mod init your-example`
3. Add dependency: `go get github.com/coded-aesthetics/bartolome-ble-toolkit@v0.1.0`
4. Import packages and use the APIs

## 📦 Package Usage

### Columbus Package
```go
import "github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/columbus"

device := columbus.NewDevice()
device.OnSignal(func(signal []byte) error {
    // Handle pen signals
    return nil
})
```

### Countries Package
```go
import "github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/countries"

country, err := countries.ResolveFromHex("1234")
if err == nil {
    fmt.Printf("Country: %s\n", country.Name)
}
```

### Timeular Package
```go
import "github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/timeular"

tracker := timeular.NewDeviceWithName("My Tracker")
tracker.OnSideChange(func(name string, side byte) error {
    fmt.Printf("Side: %d\n", side)
    return nil
})
```

## 🎯 Next Steps

1. **Start with working-columbus** for reliable Columbus Video Pen integration
2. **Explore modular examples** to understand the package architecture
3. **Build your own applications** using the toolkit packages
4. **Contribute** new device support or improvements

## 📞 Support

- Check the main README.md for comprehensive documentation
- Review package documentation in the `pkg/` directories
- Create issues on GitHub for bugs or feature requests
- Test with the `test-external` example to verify module functionality

Happy coding with Bluetooth Low Energy! 🚀