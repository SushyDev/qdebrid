package torrents

import (
	"encoding/json"
	"net/http"
	"qdebrid/qbittorrent/helpers"

	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func (module *Module) Properties(w http.ResponseWriter, r *http.Request) {
	logger := module.GetLogger()

	logger.Info("Received request for torrent properties")

	hash := r.URL.Query().Get("hash")

	torrentInfo, err := real_debrid_api.GetTorrentInfo(module.RealDebridClient, hash)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	torrentProperties, err := helpers.GetTorrentProperties(torrentInfo)

	jsonData, err := json.Marshal(torrentProperties)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
