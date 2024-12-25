package helpers

import (
	"path/filepath"
	"time"

	"github.com/sushydev/real_debrid_go/api"
)

type TorrentInfo struct {
	AddedOn            int64   `json:"added_on"`
	AmountLeft         int64   `json:"amount_left"`
	AutoTMM            bool    `json:"auto_tmm"`
	Availability       float64 `json:"availability"`
	Category           string  `json:"category"`
	Completed          int64   `json:"completed"`
	CompletionOn       int64   `json:"completion_on"`
	ContentPath        string  `json:"content_path"`
	DownloadLimit      int64   `json:"dl_limit"`
	DownloadSpeed      int64   `json:"dlspeed"`
	Downloaded         int64   `json:"downloaded"`
	DownloadedSession  int64   `json:"downloaded_session"`
	ETA                int64   `json:"eta"`
	FirstLastPiecePrio bool    `json:"f_l_piece_prio"`
	ForceStart         bool    `json:"force_start"`
	Hash               string  `json:"hash"`
	LastActivity       int64   `json:"last_activity"`
	MagnetURI          string  `json:"magnet_uri"`
	MaxRatio           float64 `json:"max_ratio"`
	MaxSeedingTime     int64   `json:"max_seeding_time"`
	Name               string  `json:"name"`
	NumComplete        int64   `json:"num_complete"`
	NumIncomplete      int64   `json:"num_incomplete"`
	NumLeechs          int64   `json:"num_leechs"`
	NumSeeds           int64   `json:"num_seeds"`
	Priority           int64   `json:"priority"`
	Progress           float64 `json:"progress"`
	Ratio              float64 `json:"ratio"`
	RatioLimit         float64 `json:"ratio_limit"`
	SavePath           string  `json:"save_path"`
	SeedingTime        int64   `json:"seeding_time"`
	SeedingTimeLimit   int64   `json:"seeding_time_limit"`
	SeenComplete       int64   `json:"seen_complete"`
	SeqDL              bool    `json:"seq_dl"`
	Size               int64   `json:"size"`
	State              string  `json:"state"`
	SuperSeeding       bool    `json:"super_seeding"`
	Tags               string  `json:"tags"`
	TimeActive         int64   `json:"time_active"`
	TotalSize          int64   `json:"total_size"`
	Tracker            string  `json:"tracker"`
	UploadLimit        int64   `json:"up_limit"`
	Uploaded           int64   `json:"uploaded"`
	UploadedSession    int64   `json:"uploaded_session"`
	UploadSpeed        int64   `json:"upspeed"`
}

func GetTorrentInfo(torrent *api.Torrent) (TorrentInfo, error) {
	state := mapRealDebridStatus(torrent.Status)

	// pathExists, err := pathExists(torrent.Filename)
	// if err != nil {
	// 	return TorrentInfo{}, err
	// }

	// if state == "pausedUP" && settings.QDebrid.ValidatePaths && !pathExists {
	// state = "checkingUP"
	// }

	addedOn, err := time.Parse(time.RFC3339Nano, torrent.Added)
	if err != nil {
		return TorrentInfo{}, err
	}

	contentPath := filepath.Join(settings.QDebrid.SavePath, torrent.ID)

	bytesTotal := int64(torrent.Bytes)
	bytesDone := int64(float64(torrent.Bytes) * (torrent.Progress / 100))
	eta := int64(60 * 60 * 24 * 365)

	torrentInfo := TorrentInfo{
		AddedOn:           addedOn.Unix(),
		AmountLeft:        bytesTotal - bytesDone,
		Category:          settings.QDebrid.CategoryName,
		ContentPath:       contentPath,
		DownloadSpeed:     int64(torrent.Speed),
		Downloaded:        bytesDone,
		DownloadedSession: bytesDone,
		Hash:              torrent.ID, // Should be hash but for /torrents/info i pass ID
		ETA:               eta,
		LastActivity:      time.Now().Unix(),
		Name:              torrent.Filename,
		NumSeeds:          int64(torrent.Seeders),
		Progress:          torrent.Progress / 100,
		SavePath:          contentPath,
		Size:              bytesTotal,
		State:             state,
		TimeActive:        time.Now().Unix() - addedOn.Unix(),
		TotalSize:         bytesTotal,
	}

	if torrent.Status == "downloading" {
		if torrent.Speed < 0 {
			torrentInfo.ETA = (bytesTotal - bytesDone) / int64(torrent.Speed)
		}
	}

	if torrent.Status == "downloaded" {
		endedOn, err := time.Parse(time.RFC3339Nano, torrent.Ended)
		if err != nil {
			return TorrentInfo{}, err
		}

		torrentInfo.SeenComplete = endedOn.Unix()
		torrentInfo.CompletionOn = endedOn.Unix()
		torrentInfo.Completed = bytesTotal
	}

	return torrentInfo, nil
}
