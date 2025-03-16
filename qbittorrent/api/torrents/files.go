package torrents

import (
	"encoding/json"
	"net/http"
	"qdebrid/cache"
	"qdebrid/debrid/client/real_debrid"
	"time"
)

// https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#get-torrent-contents

type FileRequest struct {
	Hash string `json:"hash"`
}

type fileResponse struct {
	Index        int      `json:"index"`        // File index
	Name         string   `json:"name"`         // File name (including relative path)
	Size         int      `json:"size"`         // File size (bytes)
	Progress     float64  `json:"progress"`     // File progress (percentage/100)
	Priority     priority `json:"priority"`     // File priority. See possible values here below
	IsSeed       bool     `json:"is_seed"`      // True if file is seeding/complete
	PieceRange   [2]int   `json:"piece_range"`  // The first number is the starting piece index and the second number is the ending piece index (inclusive)
	Availability float64  `json:"availability"` // Percentage of file pieces currently available (percentage/100)
}

type priority int

const (
	DoNotDownload priority = 0
	Normal        priority = 1
	High          priority = 6
	Maximal       priority = 7
)

func Files(w http.ResponseWriter, r *http.Request, c *cache.Cache) {
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

	client := real_debrid.GetClient()

	hash := r.FormValue("hash")
	torrentInfo, err := client.GetTorrentInfo(hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ding = []fileResponse{}
	for index, file := range torrentInfo.Files {
		if file.Selected == 0 {
			continue
		}

		 ding = append(ding, fileResponse{
			Index:        index,
			Name:         file.Path,
			Size:         file.Bytes,
			Progress:     torrentInfo.Progress,
			Priority:     Normal,
			IsSeed:       torrentInfo.Seeders > 0,
			PieceRange:   [2]int{0, 0},
			Availability: 100,
		})
	}

	jsonData, err := json.Marshal(ding)
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
