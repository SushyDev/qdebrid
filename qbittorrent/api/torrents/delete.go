package torrents

import (
	"net/http"
	"qdebrid/qbittorrent/helpers"

	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func (module *Module) Delete(w http.ResponseWriter, r *http.Request) {
	logger := module.GetLogger()

	logger.Info("Received request to delete torrent(s)")

	err := r.ParseForm()
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash := r.FormValue("hashes")

	torrent, err := real_debrid_api.GetTorrentInfo(module.RealDebridClient, hash)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = real_debrid_api.Delete(module.RealDebridClient, torrent.ID)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	helpers.ClearCache()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
