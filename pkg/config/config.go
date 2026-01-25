package config

import (
	"encoding/json"
	"os"
)

// configuiration struct
type ConfigFromFile struct {
	Port            int    `json:"port"`
	Strategy        string `json:"strategy"`
	HealthCheckFreq string `json:"health_check_frequency"` // e.g. "10s"
	Backends        []struct {
		URL   string `json:"url"`
		Alive bool   `json:"alive"`
		Weight int    `json:"weight"`
	} `json:"backends"`
}

// function to read the file and return the validated configuration
func LoadConfig(filename string) (*ConfigFromFile, error) {
	// open file
	file, err := os.Open(filename)
	// handle error
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// decode the file
	decoder := json.NewDecoder(file)
	// create a ConfigFromFile instance to store the file decoding
	config := &ConfigFromFile{}
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}

	return config, nil

}
