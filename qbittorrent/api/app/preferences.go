package app

import (
	"encoding/json"
)

type Config struct{}

func Preferences() []byte {
	config := Config{}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil
	}

	return jsonData
}
