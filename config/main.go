package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"reflect"
)

var config Config

func GetConfig() Config {
	if !reflect.DeepEqual(config, Config{}) {
		return config
	}

	configFile, err := os.Open("config.yml")
	if err != nil {
		fmt.Printf("Error opening config file: %v\n", err)
		panic(err)
	}

	configFileBytes, err := io.ReadAll(configFile)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		panic(err)
	}

	err = yaml.Unmarshal(configFileBytes, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling config file: %v\n", err)
		panic(err)
	}

	return config
}

func GetSettings() Settings {
	return GetConfig().Settings
}
