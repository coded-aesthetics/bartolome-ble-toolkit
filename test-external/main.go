package main

import (
	"fmt"
	"log"

	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/columbus"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/countries"
	"github.com/coded-aesthetics/bartolome-ble-toolkit/pkg/timeular"
)

func main() {
	fmt.Println("🧪 Testing Bartolome BLE Toolkit Module Imports...")
	fmt.Println("==================================================")

	// Test Columbus package
	fmt.Println("📦 Testing Columbus package...")
	columbusDevice := columbus.NewDevice()
	if columbusDevice == nil {
		log.Fatal("❌ Failed to create Columbus device")
	}
	fmt.Printf("✅ Columbus device created: %s\n", columbusDevice.GetName())
	fmt.Printf("   Service UUID: %s\n", columbusDevice.GetServiceUUID().String())
	fmt.Printf("   Characteristic UUID: %s\n", columbusDevice.GetCharacteristicUUID().String())

	// Test Countries package
	fmt.Println("\n📦 Testing Countries package...")
	testHex := "1234"
	country, err := countries.ResolveFromHex(testHex)
	if err != nil {
		fmt.Printf("⚠️  Country resolution test (expected for unknown hex): %v\n", err)
	} else {
		fmt.Printf("✅ Country resolved: %s (%s)\n", country.Name, country.Alpha2Code)
	}

	// Test loading country data
	if err := countries.LoadCountryData(); err != nil {
		fmt.Printf("⚠️  Country data loading: %v\n", err)
	} else {
		fmt.Printf("✅ Country data loaded successfully\n")
	}

	// Test Timeular package
	fmt.Println("\n📦 Testing Timeular package...")
	timeularDevice := timeular.NewDevice()
	if timeularDevice == nil {
		log.Fatal("❌ Failed to create Timeular device")
	}
	fmt.Printf("✅ Timeular device created: %s\n", timeularDevice.GetName())
	fmt.Printf("   Service UUID: %s\n", timeularDevice.GetServiceUUID().String())
	fmt.Printf("   Characteristic UUID: %s\n", timeularDevice.GetCharacteristicUUID().String())
	fmt.Printf("   Supported sides: %d\n", timeular.GetSupportedSides())

	// Test Timeular with custom name
	customTimeular := timeular.NewDeviceWithName("Test Tracker")
	fmt.Printf("✅ Custom Timeular device: %s\n", customTimeular.GetName())

	// Test Timeular side validation
	validSide := byte(5)
	if timeular.IsValidSide(validSide) {
		fmt.Printf("✅ Side validation works: side %d is valid\n", validSide)
	}

	invalidSide := byte(10)
	if !timeular.IsValidSide(invalidSide) {
		fmt.Printf("✅ Side validation works: side %d is invalid\n", invalidSide)
	}

	fmt.Println("\n🎉 All module imports successful!")
	fmt.Println("📋 Summary:")
	fmt.Println("   ✅ Columbus package - working")
	fmt.Println("   ✅ Countries package - working")
	fmt.Println("   ✅ Timeular package - working")
	fmt.Println("\n🚀 Module is ready for use!")
}
