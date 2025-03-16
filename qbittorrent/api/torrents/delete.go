package torrents

import (
	"net/http"
	"qdebrid/cache"
	"qdebrid/qbittorrent/helpers"

	"qdebrid/debrid/client/real_debrid"
)

func Delete(w http.ResponseWriter, r *http.Request, c *cache.Cache) {
	hashes, err := helpers.GetHashes(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := real_debrid.GetClient()

	for _, hash := range hashes {
		err = client.DeleteByHash(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	c.Clear()

	w.WriteHeader(http.StatusOK)
}
