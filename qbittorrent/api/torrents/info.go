package torrents

import (
	"encoding/json"
	"qdebrid/cache"
	"qdebrid/qbittorrent/helpers"
	"time"

	real_debrid "github.com/sushydev/real_debrid_go"
	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func getTorrents(client *real_debrid.Client, cacheStore *cache.Cache) (*real_debrid_api.Torrents, error) {
	cachedTorrents := cacheStore.Get("torrents")
	if cachedTorrents != nil {
		unmarshaledTorrents := &real_debrid_api.Torrents{}
		err := json.Unmarshal(cachedTorrents, unmarshaledTorrents)
		if err != nil {
			return nil, err
		}

		return unmarshaledTorrents, nil
	}

	torrents, err := real_debrid_api.GetTorrents(client)
	if err != nil {
		return nil, err
	}

	marsheledTorrents, err := json.Marshal(torrents)
	if err != nil {
		return nil, err
	}

	cacheStore.Store("torrents", cache.Entry{
		Value:      marsheledTorrents,
		Expiration: time.Now().Add(15 * time.Minute),
	})

	return torrents, nil
}

func Info(client *real_debrid.Client, cacheStore *cache.Cache) ([]byte, error) {
	torrents, err := getTorrents(client, cacheStore)
	if err != nil {
		return nil, err
	}

	torrentInfos := []helpers.TorrentInfo{}
	for _, match := range *torrents {
		torrentInfo, err := helpers.ParseTorrentInfo(match)
		if err != nil {
			return nil, err
		}

		torrentInfos = append(torrentInfos, torrentInfo)
	}

	jsonData, err := json.Marshal(torrentInfos)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
