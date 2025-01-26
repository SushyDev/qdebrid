package helpers

import (
	"os"
	"path/filepath"
	"qdebrid/config"

	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

var settings = config.GetSettings()

func pathExists(path string) (bool, error) {
	directory := filepath.Join(settings.QDebrid.SavePath, path)

	_, err := os.Stat(directory)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func mapRealDebridStatus(status string) string {
	switch status {
	case "magnet_error":
		return "error"
	case "magnet_conversion":
		return "checkingUP"
	case "waiting_files_selection":
		return "checkingUP"
	case "queued":
		return "checkingUP"
	case "downloading":
		return "downloading"
	case "downloaded":
		return "pausedUP"
	case "error":
		return "error"
	case "virus":
		return "error"
	case "compressing":
		return "checkingUP"
	case "uploading":
		return "uploading"
	case "dead":
		return "error"
	default:
		return "unknown"
	}
}

func GetTorrentIdFromHash(torrents real_debrid_api.Torrents, hash string) string {
	for _, torrent := range torrents {
		if torrent.Hash == hash {
			return torrent.ID
		}
	}

	return ""
}
