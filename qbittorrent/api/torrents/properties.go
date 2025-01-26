package torrents

import (
	"encoding/json"
	"qdebrid/qbittorrent/helpers"

	real_debrid "github.com/sushydev/real_debrid_go"
	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func Properties(client *real_debrid.Client, hash string) ([]byte, error) {
	torrents, err := real_debrid_api.GetTorrents(client)
	if err != nil {
		return nil, err
	}

	id := helpers.GetTorrentIdFromHash(*torrents, hash)

	torrentInfo, err := real_debrid_api.GetTorrentInfo(client, id)
	if err != nil {
		return nil, err
	}

	torrentProperties, err := helpers.GetTorrentProperties(torrentInfo)

	jsonData, err := json.Marshal(torrentProperties)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
