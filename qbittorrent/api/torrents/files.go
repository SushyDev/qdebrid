package torrents

import (
	"encoding/json"

	real_debrid "github.com/sushydev/real_debrid_go"
	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

// https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#get-torrent-contents

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

func Files(client *real_debrid.Client, hash string) ([]byte, error) {
	torrentInfo, err := real_debrid_api.GetTorrentInfo(client, hash)
	if err != nil {
		return nil, err
	}

	var files = []fileResponse{}
	for index, torrentFile := range torrentInfo.Files {
		file := fileResponse{
			Index:        index,
			Name:         torrentFile.Path,
			Size:         torrentFile.Bytes,
			Progress:     float64(torrentInfo.Progress),
			Priority:     Normal,
			IsSeed:       torrentInfo.Seeders > 0,
			PieceRange:   [2]int{0, 0},
			Availability: 100,
		}

		files = append(files, file)
	}

	jsonData, err := json.Marshal(files)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}
