package examples

import (
	"bluetooth_connector"
	"columbus"
	"country_resolver"
	"fmt"
	"net/http"
	"net/url"
	"utils"
)

type CountryAndCategory struct {
	Country    string
	Category   int
	Category_2 int
}

func Init_Columbus_And_Explorer_Pen() {
	channel_country_and_category := make(chan CountryAndCategory)
	go bluetooth_connector.Connect_And_Reconnect_To_Devices([]bluetooth_connector.Device_To_Discover{
		columbus.Columbus_Device,
	}, Listen_To_Bluetooth_Events(channel_country_and_category))

	for country_and_category := range channel_country_and_category {
		Send_Request_To_Play_By_Country_And_Category(country_and_category)
	}
}

func Send_Request_To_Play_By_Country_And_Category(country_and_category CountryAndCategory) {
	client := &http.Client{}
	data := url.Values{}
	data.Set("country", country_and_category.Country)
	data.Set("category", fmt.Sprintf("%d", country_and_category.Category))
	data.Set("category_2", fmt.Sprintf("%d", country_and_category.Category_2))

	req, err := http.NewRequest("GET", "http://localhost:8888/play-by-country", nil)

	if err != nil {
		fmt.Printf("request could not be created %s\n", err)
	}

	req.URL.RawQuery = data.Encode()

	_, err = client.Do(req)
	if err != nil {
		fmt.Printf("%s\n", err)
	}
}

func Listen_To_Bluetooth_Events(channel_country_and_category chan CountryAndCategory) func(discovered_characteristics []bluetooth_connector.Discovered_Characteristic, stop_channel chan bool) {
	return func(discovered_characteristics []bluetooth_connector.Discovered_Characteristic, stop_channel chan bool) {
		var Pen bluetooth_connector.Discovered_Characteristic
		var hasPen bool

		// Find Columbus pen
		penDevices := utils.Filter_Array(discovered_characteristics, func(char bluetooth_connector.Discovered_Characteristic) bool {
			return char.Name == columbus.Columbus_Device_Name
		})
		if len(penDevices) > 0 {
			Pen = penDevices[0]
			hasPen = true
		}

		if !hasPen {
			fmt.Printf("❌ Columbus pen not found in connected devices\n")
			return
		}

		fmt.Printf("✅ Columbus pen connected and ready\n")

		// Set default values for missing timeular devices
		timeular_side := byte(1)   // Default category
		timeular_side_2 := byte(1) // Default category 2
		for {
			select {
			case <-stop_channel:
				return
			case pen_signal := <-Pen.Channel:
				fmt.Printf("🖊️  Columbus signal received: [%x]\n", pen_signal)

				// Check if signal is valid (not empty and has minimum required length)
				if len(pen_signal) == 0 {
					fmt.Printf("⚠️  Empty signal received - device may be disconnecting\n")
					continue
				}

				signal_hex := fmt.Sprintf("%x", pen_signal)
				if len(signal_hex) < 14 {
					fmt.Printf("⚠️  Signal too short (%d chars): %s - ignoring\n", len(signal_hex), signal_hex)
					continue
				}

				country, err := country_resolver.Resolve_By_Bluetooth_Signal(signal_hex)

				if err != nil {
					fmt.Printf("❌ Could not resolve country: %s\n", err)
				} else {
					fmt.Printf("✅ Columbus country: [%s]\n", country.Name)
					fmt.Printf("📊 Using default categories: %d, %d (Timeular devices not connected)\n", int(timeular_side), int(timeular_side_2))

					channel_country_and_category <- CountryAndCategory{
						Country:    country.Name,
						Category:   int(timeular_side),
						Category_2: int(timeular_side_2),
					}
				}
				fmt.Println("")
			}
		}
	}
}
