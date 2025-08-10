// Package countries provides country resolution functionality for Columbus Video Pen signals.
// It maps hex codes from pen signals to country information.
package countries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

// Country represents country information with codes and geographic data
type Country struct {
	Name                   string `json:"name"`
	Alpha2Code             string `json:"alpha_2_code"`
	Alpha3Code             string `json:"alpha_3_code"`
	CountryCode            int    `json:"country_code"`
	ISO3166_2              string `json:"iso_3166_2"`
	Region                 string `json:"region"`
	SubRegion              string `json:"sub_region"`
	IntermediateRegion     string `json:"intermediate_region"`
	RegionCode             int    `json:"region_code"`
	SubRegionCode          int    `json:"sub_region_code"`
	IntermediateRegionCode int    `json:"intermediate_region_code,omitempty"`
	GlobeHex               string `json:"globe_hex"`
}

// Resolver handles country resolution from various input formats
type Resolver struct {
	countries    []Country
	hexToCountry map[string]*Country
	loaded       bool
}

// NewResolver creates a new country resolver instance
func NewResolver() *Resolver {
	return &Resolver{
		hexToCountry: make(map[string]*Country),
	}
}

// LoadCountryData loads country data from the JSON file
func (r *Resolver) LoadCountryData() error {
	if r.loaded {
		return nil
	}

	// Get the path to the country data file
	dataPath, err := r.getCountryDataPath()
	if err != nil {
		return fmt.Errorf("failed to locate country data: %v", err)
	}

	// Read the JSON file
	content, err := ioutil.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("failed to read country data file: %v", err)
	}

	// Parse JSON
	if err := json.Unmarshal(content, &r.countries); err != nil {
		return fmt.Errorf("failed to parse country data: %v", err)
	}

	// Build hex lookup map
	r.buildHexLookupMap()
	r.loaded = true

	return nil
}

// getCountryDataPath attempts to find the country_codes.json file
func (r *Resolver) getCountryDataPath() (string, error) {
	// Get the current file's directory
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not determine package location")
	}

	packageDir := filepath.Dir(currentFile)

	// Try various possible locations for the country data file
	possiblePaths := []string{
		filepath.Join(packageDir, "country_codes.json"),
		filepath.Join(packageDir, "..", "..", "country_resolver", "country_codes.json"),
		"./country_resolver/country_codes.json",
		"./pkg/countries/country_codes.json",
		"./country_codes.json",
	}

	for _, path := range possiblePaths {
		if _, err := ioutil.ReadFile(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("country_codes.json not found in any expected location")
}

// buildHexLookupMap creates a fast lookup map for hex codes
func (r *Resolver) buildHexLookupMap() {
	for i := range r.countries {
		country := &r.countries[i]
		if country.GlobeHex != "" {
			// Store both uppercase and lowercase versions
			r.hexToCountry[strings.ToUpper(country.GlobeHex)] = country
			r.hexToCountry[strings.ToLower(country.GlobeHex)] = country
		}
	}
}

// ResolveFromSignal resolves country from a Columbus pen signal
func (r *Resolver) ResolveFromSignal(signal []byte) (*Country, error) {
	if err := r.LoadCountryData(); err != nil {
		return nil, err
	}

	// Convert signal to hex string
	hexStr := fmt.Sprintf("%x", signal)

	// Check if hex string is long enough for country extraction
	if len(hexStr) < 14 {
		return nil, fmt.Errorf("signal too short for country extraction: %s (length: %d)", hexStr, len(hexStr))
	}

	// Extract country hex (positions 10-13 in hex string)
	countryHex := hexStr[10:14]
	return r.ResolveFromHex(countryHex)
}

// ResolveFromHex resolves country from a hex code string
func (r *Resolver) ResolveFromHex(hex string) (*Country, error) {
	if err := r.LoadCountryData(); err != nil {
		return nil, err
	}

	// Clean the hex string
	hex = strings.TrimSpace(hex)
	if hex == "" {
		return nil, fmt.Errorf("empty hex code")
	}

	// Try to find the country
	if country, exists := r.hexToCountry[hex]; exists {
		return country, nil
	}

	// Try uppercase version
	if country, exists := r.hexToCountry[strings.ToUpper(hex)]; exists {
		return country, nil
	}

	return nil, fmt.Errorf("country not found for hex code: %s", hex)
}

// ResolveFromCountryCode resolves country from a numeric country code
func (r *Resolver) ResolveFromCountryCode(code int) (*Country, error) {
	if err := r.LoadCountryData(); err != nil {
		return nil, err
	}

	for i := range r.countries {
		if r.countries[i].CountryCode == code {
			return &r.countries[i], nil
		}
	}

	return nil, fmt.Errorf("country not found for code: %d", code)
}

// ResolveFromAlpha2Code resolves country from a 2-letter country code (e.g., "US")
func (r *Resolver) ResolveFromAlpha2Code(code string) (*Country, error) {
	if err := r.LoadCountryData(); err != nil {
		return nil, err
	}

	code = strings.ToUpper(strings.TrimSpace(code))
	for i := range r.countries {
		if r.countries[i].Alpha2Code == code {
			return &r.countries[i], nil
		}
	}

	return nil, fmt.Errorf("country not found for alpha-2 code: %s", code)
}

// GetAllCountries returns all loaded countries
func (r *Resolver) GetAllCountries() ([]Country, error) {
	if err := r.LoadCountryData(); err != nil {
		return nil, err
	}

	result := make([]Country, len(r.countries))
	copy(result, r.countries)
	return result, nil
}

// GetCountriesByRegion returns all countries in a specific region
func (r *Resolver) GetCountriesByRegion(region string) ([]Country, error) {
	if err := r.LoadCountryData(); err != nil {
		return nil, err
	}

	var result []Country
	for _, country := range r.countries {
		if strings.EqualFold(country.Region, region) {
			result = append(result, country)
		}
	}

	return result, nil
}

// ValidateSignalFormat checks if a signal has the expected format for country resolution
func ValidateSignalFormat(signal []byte) error {
	if len(signal) == 0 {
		return fmt.Errorf("empty signal")
	}

	hexStr := fmt.Sprintf("%x", signal)
	if len(hexStr) < 14 {
		return fmt.Errorf("signal too short: expected at least 14 hex characters, got %d", len(hexStr))
	}

	return nil
}

// Package-level convenience functions using a default resolver
var defaultResolver = NewResolver()

// ResolveFromSignal is a convenience function using the default resolver
func ResolveFromSignal(signal []byte) (*Country, error) {
	return defaultResolver.ResolveFromSignal(signal)
}

// ResolveFromHex is a convenience function using the default resolver
func ResolveFromHex(hex string) (*Country, error) {
	return defaultResolver.ResolveFromHex(hex)
}

// LoadCountryData loads country data using the default resolver
func LoadCountryData() error {
	return defaultResolver.LoadCountryData()
}

// Legacy function for backward compatibility
func Resolve_By_Bluetooth_Signal(bluetooth_signal string) (*Country, error) {
	// This function expects a hex string, not bytes
	if len(bluetooth_signal) < 14 {
		return nil, fmt.Errorf("signal too short for country extraction: %s (length: %d)", bluetooth_signal, len(bluetooth_signal))
	}

	hex_part := bluetooth_signal[10:14]
	return ResolveFromHex(hex_part)
}

// Legacy function for backward compatibility
func Resolve_By_Country_Hex(country_hex string) (*Country, error) {
	return ResolveFromHex(country_hex)
}

// Legacy type for backward compatibility
type Country_Code = Country
