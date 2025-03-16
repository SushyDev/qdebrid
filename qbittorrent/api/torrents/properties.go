package torrents

import (
	"encoding/json"
	"net/http"
	"qdebrid/cache"
	"qdebrid/qbittorrent/helpers"
	"time"

	"qdebrid/debrid/client/real_debrid"
)

func Properties(w http.ResponseWriter, r *http.Request, c *cache.Cache) {
	cacheKey, err := cache.GetCacheKeyByRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cachedData := c.Get(cacheKey)
	if cachedData != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(cachedData)
		return
	}

	hash, err := helpers.GetHash(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := real_debrid.GetClient()

	torrentInfo, err := client.GetTorrentInfoByHash(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	properties, err := helpers.GetTorrentProperties(torrentInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(properties)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.Store(cacheKey, cache.Entry{
		Value:      jsonData,
		Expiration: time.Now().Add(15 * time.Minute),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
