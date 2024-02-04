package app

import (
	"encoding/json"
	"net/http"
)

func Version(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("2.0"))
}

func Preferences(w http.ResponseWriter, r *http.Request) {
	config := QBitTorrentConfig{}

	jsonData, err := json.Marshal(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
