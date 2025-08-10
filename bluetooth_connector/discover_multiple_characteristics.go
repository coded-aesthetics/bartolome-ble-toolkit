package bluetooth_connector

import (
	"context"
	"fmt"
	"sync"
	"time"
	"utils"

	"tinygo.org/x/bluetooth"
)

type Device_To_Discover struct {
	Name                              string
	ServiceUUID                       bluetooth.UUID
	CharacteristicUUID                bluetooth.UUID
	Establish_Characteristic_Listener func(bluetooth.DeviceCharacteristic) (chan []byte, func(), error)
}

type Discovered_Characteristic struct {
	Name                            string
	Address                         bluetooth.Address
	Device                          *bluetooth.Device
	Service                         *bluetooth.DeviceService
	Characteristic                  *bluetooth.DeviceCharacteristic
	Channel                         chan []byte
	Disable_Characteristic_Listener func()
}

func New_Discovered_Characteristic(Name string, Address bluetooth.Address, Channel chan []byte, Service *bluetooth.DeviceService,
	Characteristic *bluetooth.DeviceCharacteristic, Disable_Characteristic_Listener func(), Device *bluetooth.Device) *Discovered_Characteristic {
	discovered_characteristic := new(Discovered_Characteristic)
	discovered_characteristic.Name = Name
	discovered_characteristic.Address = Address
	discovered_characteristic.Characteristic = Characteristic
	discovered_characteristic.Service = Service
	discovered_characteristic.Channel = Channel
	discovered_characteristic.Device = Device
	discovered_characteristic.Disable_Characteristic_Listener = Disable_Characteristic_Listener

	return discovered_characteristic
}

var (
	adapterEnabled bool
	adapterMutex   sync.Mutex
)

func Discover_Multiple_Characteristics(devices_to_discover []Device_To_Discover) (chan *Discovered_Characteristic, chan error, error) {
	channel := make(chan *Discovered_Characteristic)
	err_channel := make(chan error)
	adapter := bluetooth.DefaultAdapter

	// Enable BLE interface only once with proper synchronization
	adapterMutex.Lock()
	if !adapterEnabled {
		fmt.Println("Enabling BLE adapter...")
		err := adapter.Enable()
		if err != nil {
			adapterMutex.Unlock()
			return nil, nil, fmt.Errorf("could not enable the BLE stack: %v", err)
		}
		adapterEnabled = true
		fmt.Println("✅ BLE adapter enabled successfully")

		// Give macOS time to initialize properly
		time.Sleep(2 * time.Second)
	} else {
		fmt.Println("BLE adapter already enabled")
	}
	adapterMutex.Unlock()

	// Make a copy of devices to discover
	devices_copy := make([]Device_To_Discover, len(devices_to_discover))
	copy(devices_copy, devices_to_discover)

	go run_scan(channel, err_channel, adapter, devices_copy)

	return channel, err_channel, nil
}

func run_scan(channel chan *Discovered_Characteristic, err_channel chan error, adapter *bluetooth.Adapter, devices_to_discover []Device_To_Discover) {
	discovered_characteristics := make([]Discovered_Characteristic, 0)

	fmt.Println("[Scanning for devices]")
	fmt.Println("")

	// Create context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Monitor for timeout
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded && len(discovered_characteristics) < len(devices_to_discover) {
			err_channel <- fmt.Errorf("scan timeout: only found %d/%d devices", len(discovered_characteristics), len(devices_to_discover))
		}
	}()

	err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		// Check if this is a device we're looking for
		discovered_device, index_of_discovered_device := is_device_to_discover(devices_to_discover, result)
		if discovered_device == nil {
			return
		}

		// Check if we already found this device
		found_characteristics := utils.Filter_Array(discovered_characteristics, func(discovered_characteristic Discovered_Characteristic) bool {
			return discovered_characteristic.Address.String() == result.Address.String()
		})

		if len(found_characteristics) != 0 {
			return
		}

		// CRITICAL: Stop scanning immediately before attempting connection (macOS requirement)
		fmt.Printf("📱 Found target device %s, stopping scan to connect...\n", discovered_device.Name)
		adapter.StopScan()

		// Process this device connection in a separate goroutine
		go process_device_connection(result, *discovered_device, index_of_discovered_device,
			&discovered_characteristics, devices_to_discover, channel, err_channel,
			adapter, ctx)
	})

	if err != nil {
		err_channel <- fmt.Errorf("could not start a scan: %v", err)
	}
}

func process_device_connection(result bluetooth.ScanResult, discovered_device Device_To_Discover,
	index_of_discovered_device int, discovered_characteristics *[]Discovered_Characteristic,
	devices_to_discover []Device_To_Discover, channel chan *Discovered_Characteristic,
	err_channel chan error, adapter *bluetooth.Adapter, ctx context.Context) {

	// Brief delay to ensure scan is fully stopped
	time.Sleep(500 * time.Millisecond)

	// Attempt connection
	device, err := connect_to_device_with_retry(adapter, result, 2)
	if err != nil {
		fmt.Printf("❌ Failed to connect to %s: %s\n", discovered_device.Name, err.Error())
		restart_scan_after_failure(adapter, ctx, devices_to_discover, *discovered_characteristics, channel, err_channel)
		return
	}

	// Discover service and characteristic
	service, characteristic, err := discover_service_and_characteristic(*device, discovered_device)
	if err != nil {
		fmt.Printf("❌ Failed to discover service/characteristic for %s: %s\n", discovered_device.Name, err.Error())
		device.Disconnect()
		restart_scan_after_failure(adapter, ctx, devices_to_discover, *discovered_characteristics, channel, err_channel)
		return
	}

	// Establish characteristic listener
	Chan, Disable_Characteristic_Listener, err := discovered_device.Establish_Characteristic_Listener(*characteristic)
	if err != nil {
		fmt.Printf("❌ Failed to establish listener for %s: %s\n", discovered_device.Name, err.Error())
		device.Disconnect()
		restart_scan_after_failure(adapter, ctx, devices_to_discover, *discovered_characteristics, channel, err_channel)
		return
	}

	fmt.Printf("✅ Successfully connected to %s\n", discovered_device.Name)
	fmt.Println("")

	discovered_characteristic := New_Discovered_Characteristic(
		discovered_device.Name,
		result.Address,
		Chan,
		service,
		characteristic,
		Disable_Characteristic_Listener,
		device,
	)
	*discovered_characteristics = append(*discovered_characteristics, *discovered_characteristic)

	channel <- discovered_characteristic

	// Remove this device from the list to discover
	updated_devices := remove(devices_to_discover, index_of_discovered_device)

	if len(updated_devices) == 0 {
		fmt.Println("✅ All devices discovered!")
		return
	}

	// Continue scanning for remaining devices after a brief delay
	time.Sleep(1 * time.Second)
	restart_scan_for_remaining(adapter, ctx, updated_devices, *discovered_characteristics, channel, err_channel)
}

func restart_scan_after_failure(adapter *bluetooth.Adapter, ctx context.Context, devices_to_discover []Device_To_Discover, discovered_characteristics []Discovered_Characteristic, channel chan *Discovered_Characteristic, err_channel chan error) {
	// Check if context is still valid
	select {
	case <-ctx.Done():
		return
	default:
		// Wait a bit longer before retrying scan after failure
		time.Sleep(3 * time.Second)
		run_scan(channel, err_channel, adapter, devices_to_discover)
	}
}

func restart_scan_for_remaining(adapter *bluetooth.Adapter, ctx context.Context, devices_to_discover []Device_To_Discover, discovered_characteristics []Discovered_Characteristic, channel chan *Discovered_Characteristic, err_channel chan error) {
	// Check if context is still valid
	select {
	case <-ctx.Done():
		return
	default:
		run_scan(channel, err_channel, adapter, devices_to_discover)
	}
}

func connect_to_device_with_retry(adapter *bluetooth.Adapter, scanResult bluetooth.ScanResult, maxRetries int) (*bluetooth.Device, error) {
	var device bluetooth.Device
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		name := scanResult.LocalName()
		if name == "" {
			name = "Unknown"
		}
		fmt.Printf("Connection attempt %d/%d to %s...\n", attempt, maxRetries, name)

		device, err = connect_to_device_immediately(adapter, scanResult)
		if err == nil {
			return &device, nil
		}

		fmt.Printf("❌ Attempt %d failed: %s\n", attempt, err.Error())

		if attempt < maxRetries {
			// Use fixed delay instead of exponential backoff for BLE
			delay := 3 * time.Second
			fmt.Printf("Retrying in %v...\n", delay)
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %v", maxRetries, err)
}

func connect_to_device_immediately(adapter *bluetooth.Adapter, scanResult bluetooth.ScanResult) (bluetooth.Device, error) {
	name := scanResult.LocalName()
	if name == "" {
		name = "Unknown"
	}

	fmt.Printf("   → Connecting to %s [%s]...\n", name, scanResult.Address.String())

	// Direct connection without goroutine for better reliability on macOS
	device, err := adapter.Connect(scanResult.Address, bluetooth.ConnectionParams{
		ConnectionTimeout: bluetooth.NewDuration(10 * time.Second),
	})

	if err != nil {
		return bluetooth.Device{}, fmt.Errorf("connection failed: %v", err)
	}

	fmt.Printf("   ✅ Connected to %s\n", name)
	return device, nil
}

func discover_service_and_characteristic(device bluetooth.Device, discovered_device Device_To_Discover) (*bluetooth.DeviceService, *bluetooth.DeviceCharacteristic, error) {
	fmt.Printf("Discovering services for %s...\n", discovered_device.Name)

	// Create context with timeout for service discovery
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	serviceChan := make(chan []bluetooth.DeviceService, 1)
	errChan := make(chan error, 1)

	go func() {
		services, err := device.DiscoverServices([]bluetooth.UUID{discovered_device.ServiceUUID})
		if err != nil {
			errChan <- err
			return
		}
		serviceChan <- services
	}()

	var services []bluetooth.DeviceService
	select {
	case services = <-serviceChan:
		fmt.Printf("✅ Services discovered for %s\n", discovered_device.Name)
	case err := <-errChan:
		return nil, nil, fmt.Errorf("failed to discover services: %v", err)
	case <-ctx.Done():
		return nil, nil, fmt.Errorf("service discovery timeout")
	}

	if len(services) == 0 {
		return nil, nil, fmt.Errorf("no services found")
	}
	service := services[0]

	fmt.Printf("Discovering characteristics for %s...\n", discovered_device.Name)

	// Create context with timeout for characteristic discovery
	ctx2, cancel2 := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel2()

	charChan := make(chan []bluetooth.DeviceCharacteristic, 1)
	errChan2 := make(chan error, 1)

	go func() {
		chars, err := service.DiscoverCharacteristics([]bluetooth.UUID{discovered_device.CharacteristicUUID})
		if err != nil {
			errChan2 <- err
			return
		}
		charChan <- chars
	}()

	var chars []bluetooth.DeviceCharacteristic
	select {
	case chars = <-charChan:
		fmt.Printf("✅ Characteristics discovered for %s\n", discovered_device.Name)
	case err := <-errChan2:
		return nil, nil, fmt.Errorf("failed to discover characteristics: %v", err)
	case <-ctx2.Done():
		return nil, nil, fmt.Errorf("characteristic discovery timeout")
	}

	if len(chars) == 0 {
		return nil, nil, fmt.Errorf("no characteristics found")
	}

	return &service, &chars[0], nil
}

func is_device_to_discover(devices_to_discover []Device_To_Discover, result bluetooth.ScanResult) (*Device_To_Discover, int) {
	deviceName := result.LocalName()

	// First try to match by service UUID (preferred method)
	for index, device_to_discover := range devices_to_discover {
		service_uuid := device_to_discover.ServiceUUID
		if result.AdvertisementPayload.HasServiceUUID(service_uuid) {
			return &device_to_discover, index
		}
	}

	// Fallback: match by device name (needed for macOS compatibility)
	// Some devices don't advertise their service UUIDs but can be identified by name
	if deviceName != "" {
		for index, device_to_discover := range devices_to_discover {
			if device_to_discover.Name == deviceName {
				fmt.Printf("📝 Found device by name: %s (service UUID not in advertisement)\n", deviceName)
				return &device_to_discover, index
			}
		}
	}

	return nil, -1
}

func remove[T any](s []T, i int) []T {
	if i < 0 || i >= len(s) {
		return s
	}
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
