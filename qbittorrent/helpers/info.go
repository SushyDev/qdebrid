package helpers

import (
	"math"
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
	Eta                int64   `json:"eta"`
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

func ParseTorrentInfo(torrent *api.Torrent) (TorrentInfo, error) {
	state := mapRealDebridStatus(torrent.Status)

	if settings.QDebrid.ValidatePaths {
		pathExists, err := pathExists(torrent.ID)
		if err != nil {
			pathExists = false
		}

		if state == "pausedUP" && !pathExists {
			state = "missingFiles"
		}
	}

	contentPath := filepath.Join(settings.QDebrid.SavePath, torrent.ID)

	bytesTotal := int64(torrent.Bytes)
	bytesDone := int64(float64(torrent.Bytes) * (torrent.Progress / 100))

	eta := int64(60 * 60 * 24 * 365)
	if torrent.Status == "downloading" {
		if torrent.Speed < 0 {
			eta = (bytesTotal - bytesDone) / int64(torrent.Speed)
		}
	}

	progress := float64(torrent.Progress) / 100

	torrentInfo := TorrentInfo{
		Hash:         torrent.ID,
		Name:         torrent.Filename,
		Size:         int64(torrent.Bytes),
		Progress:     progress,
		Eta:          eta,
		State:        state,
		Category:     settings.QDebrid.CategoryName,
		SavePath:     contentPath,
		ContentPath:  contentPath,
		Ratio:        math.MaxInt64,
		RatioLimit:   -2,
		LastActivity: time.Now().Unix(),
	}

	if torrent.Status == "downloaded" {
		endedOn, err := time.Parse(time.RFC3339Nano, torrent.Ended)
		if err != nil {
			return TorrentInfo{}, err
		}

		torrentInfo.LastActivity = endedOn.Unix()
	}

	return torrentInfo, nil
}
