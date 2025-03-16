package torrents

import (
	"encoding/json"
	"net/http"
	"qdebrid/cache"
	"qdebrid/qbittorrent/helpers"
	"qdebrid/servarr"
	"strings"
	"time"

	"qdebrid/debrid/client/real_debrid"

	"github.com/sushydev/real_debrid_go/api"
)

func Info(w http.ResponseWriter, r *http.Request, c *cache.Cache) {
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

	host, token, err := helpers.DecodeAuthHeader(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := real_debrid.GetClient()

	torrents, err := client.GetTorrents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	history, err := servarr.GetHistory(host, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var matchedTorrents []*api.Torrent

	for _, torrent := range *torrents {
		for _, record := range history {
			if strings.EqualFold(record.DownloadID, torrent.Hash) {
				matchedTorrents = append(matchedTorrents, torrent)
				break
			}
		}
	}

	torrentInfos := []helpers.TorrentInfo{}
	for _, match := range matchedTorrents {
		torrentInfo, err := helpers.ParseTorrentInfo(match)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		torrentInfos = append(torrentInfos, torrentInfo)
	}

	jsonData, err := json.Marshal(torrentInfos)
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
