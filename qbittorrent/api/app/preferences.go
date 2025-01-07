package app

import (
	"encoding/json"
)

type Config struct{
	DhtEnabled bool `json:"dht"`
}

func Preferences() []byte {
	config := Config{
		// Allow magnets without trackers
		DhtEnabled: true,
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil
	}

	return jsonData
}
