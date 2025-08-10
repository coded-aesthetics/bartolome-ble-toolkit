package country_resolver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

type Country_Code struct {
	Name                     string
	Alpha_2                  string
	Alpha_3                  string
	Country_code             int
	Iso_3166_2               string
	Region                   string
	Sub_region               string
	Intermediate_region      string
	Region_code              int
	Sub_region_code          int
	Intermediate_region_code int `json:"intermediate_region_code,omitempty"`
	Globe_hex                string
}

func Resolve_By_Bluetooth_Signal(bluetooth_signal string) (*Country_Code, error) {
	hex_part := bluetooth_signal[10:14]
	return Resolve_By_Country_Hex(hex_part)
}

func Resolve_By_Country_Hex(country_hex string) (*Country_Code, error) {
	content, err := ioutil.ReadFile("./country_resolver/country_codes.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
		return nil, err
	}

	// Now let's unmarshall the data into `payload`
	var countries []Country_Code
	err = json.Unmarshal(content, &countries)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
		return nil, err
	}

	country, err := lookup_by_country_hex(countries, country_hex)

	if err != nil {
		return nil, err
	}

	return country, nil
}

func lookup_by_country_hex(countries []Country_Code, country_hex string) (*Country_Code, error) {
	for _, country := range countries {
		if strings.EqualFold(country_hex, country.Globe_hex) {
			return &country, nil
		}
	}

	return nil, fmt.Errorf("country with hex code %s not found", country_hex)
}
