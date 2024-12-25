package app

import (
	"encoding/json"
	"net/http"
)

type Config struct{}

func (*Module) Preferences(w http.ResponseWriter, r *http.Request) {
	config := Config{}

	jsonData, err := json.Marshal(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
