package torrents

import (
	real_debrid "github.com/sushydev/real_debrid_go"
	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func Delete(client *real_debrid.Client, hash string) error {
	torrent, err := real_debrid_api.GetTorrentInfo(client, hash)
	if err != nil {
		return err
	}

	err = real_debrid_api.Delete(client, torrent.ID)
	if err != nil {
		return err
	}

	return nil
}
