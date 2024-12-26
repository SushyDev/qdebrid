package helpers

import (
	"sync"
	"time"

	real_debrid "github.com/sushydev/real_debrid_go"
	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

type cacheEntry struct {
	data      *real_debrid_api.TorrentInfo
	timestamp time.Time
}

var (
	cache    = make(map[string]cacheEntry)
	cacheMu  sync.Mutex
	cacheTTL = 15 * time.Minute // Time to live for cache entries
)

func GetTorrentInfoWithCache(client *real_debrid.Client, id string) (*real_debrid_api.TorrentInfo, error) {
	cacheMu.Lock()
	entry, found := cache[id]
	cacheMu.Unlock()

	if found && time.Since(entry.timestamp) < cacheTTL {
		return entry.data, nil
	}

	torrentInfo, err := real_debrid_api.GetTorrentInfo(client, id)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	cache[id] = cacheEntry{
		data:      torrentInfo,
		timestamp: time.Now(),
	}
	cacheMu.Unlock()

	return torrentInfo, nil
}

func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	cache = make(map[string]cacheEntry)
}
