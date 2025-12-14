package qbittorrent

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sushydev/real_debrid_go/api"
	"qdebrid/internal/config"
)

// TorrentInfo represents a qBittorrent torrent info response
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
	Label              string  `json:"label"` // For backwards compatibility with older Servarr
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

// TorrentProperties represents qBittorrent torrent properties
type TorrentProperties struct {
	AddedOn                int64   `json:"added_on"`
	CompletionOn           int64   `json:"completion_on"`
	CreatedBy              string  `json:"created_by"`
	CreationDate           int64   `json:"creation_date"`
	Comment                string  `json:"comment"`
	DownloadLimit          int64   `json:"dl_limit"`
	DownloadSpeed          int64   `json:"dl_speed"`
	DownloadSpeedAvg       int64   `json:"dl_speed_avg"`
	Eta                    int64   `json:"eta"`
	LastSeen               int64   `json:"last_seen"`
	NbConnections          int64   `json:"nb_connections"`
	NbConnectionsLimit     int64   `json:"nb_connections_limit"`
	Peers                  int64   `json:"peers"`
	PeersTotal             int64   `json:"peers_total"`
	PieceSize              int64   `json:"piece_size"`
	PiecesHave             int64   `json:"pieces_have"`
	PiecesNum              int64   `json:"pieces_num"`
	Reannounce             int64   `json:"reannounce"`
	SavePath               string  `json:"save_path"`
	SeedingTime            int64   `json:"seeding_time"`
	Seeds                  int64   `json:"seeds"`
	SeedsTotal             int64   `json:"seeds_total"`
	ShareRatio             float64 `json:"share_ratio"`
	TimeElapsed            int64   `json:"time_elapsed"`
	TotalDownloaded        int64   `json:"total_downloaded"`
	TotalDownloadedSession int64   `json:"total_downloaded_session"`
	TotalSize              int64   `json:"total_size"`
	TotalUploaded          int64   `json:"total_uploaded"`
	TotalUploadedSession   int64   `json:"total_uploaded_session"`
	TotalWasted            int64   `json:"total_wasted"`
	UploadLimit            int64   `json:"up_limit"`
	UploadSpeed            int64   `json:"up_speed"`
	UploadSpeedAvg         int64   `json:"up_speed_avg"`
}

// TorrentFile represents a file in a torrent
type TorrentFile struct {
	Index        int     `json:"index"`
	Name         string  `json:"name"`
	Size         int64   `json:"size"`
	Progress     float64 `json:"progress"`
	Priority     int     `json:"priority"`
	IsSeed       bool    `json:"is_seed"`
	PieceRange   []int   `json:"piece_range"`
	Availability float64 `json:"availability"`
}

// Category represents a torrent category
type Category struct {
	Name     string `json:"name"`
	SavePath string `json:"savePath"`
}

// Preferences represents qBittorrent preferences
type Preferences struct {
	SavePath string `json:"save_path,omitempty"` // omitempty so it's not included if empty
	Dht      bool   `json:"dht"`                 // Allow magnets without trackers
}

// ParseAuthHeader extracts Servarr host and API key from Basic Auth header
// The username field contains the Servarr host, password contains the API key
func ParseAuthHeader(r *http.Request) (host string, apiKey string, err error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", "", fmt.Errorf("authorization header missing")
	}

	// Remove "Basic " prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return "", "", fmt.Errorf("invalid authorization header format")
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 encoding: %w", err)
	}

	// Split username:password
	credentials := string(decoded)
	colonIndex := strings.LastIndex(credentials, ":")
	if colonIndex == -1 {
		return "", "", fmt.Errorf("invalid credentials format")
	}

	host = credentials[:colonIndex]
	apiKey = credentials[colonIndex+1:]

	return host, apiKey, nil
}

// ConvertRealDebridToTorrentInfo converts Real-Debrid torrent to qBittorrent format
func ConvertRealDebridToTorrentInfo(rdTorrent *api.Torrent, cfg *config.QBittorrentConfig, state string) TorrentInfo {
	// SavePath is the base directory where torrents are saved
	// ContentPath is the full path including the torrent's subfolder (torrent hash)
	savePath := cfg.SavePath
	// Use torrent hash for content path, fallback to ID if hash is empty
	hashOrID := rdTorrent.Hash
	if hashOrID == "" {
		hashOrID = rdTorrent.ID
	}
	contentPath := filepath.Join(cfg.SavePath, hashOrID)

	bytesTotal := int64(rdTorrent.Bytes)
	bytesDone := int64(float64(rdTorrent.Bytes) * (rdTorrent.Progress / 100))
	amountLeft := bytesTotal - bytesDone

	// Calculate ETA
	eta := int64(8640000) // Default: very large number (100 days in seconds)
	if rdTorrent.Status == "downloading" && rdTorrent.Speed > 0 {
		eta = amountLeft / int64(rdTorrent.Speed)
	}

	progress := rdTorrent.Progress / 100.0

	// Ensure we have a name (use ID as fallback if filename is empty)
	name := rdTorrent.Filename
	if name == "" {
		name = rdTorrent.ID
	}

	// Return the actual infohash (torrent.Hash), not Real-Debrid ID
	// This matches old implementation and is what Radarr expects
	hash := rdTorrent.Hash
	if hash == "" {
		hash = rdTorrent.ID
	}

	info := TorrentInfo{
		Hash:              hash,
		Name:              name,
		MagnetURI:         fmt.Sprintf("magnet:?xt=urn:btih:%s", rdTorrent.Hash),
		Size:              bytesTotal,
		Progress:          progress,
		Downloaded:        bytesDone,
		DownloadedSession: bytesDone,
		AmountLeft:        amountLeft,
		Eta:               eta,
		State:             state,
		Category:          cfg.CategoryName,
		Label:             cfg.CategoryName, // For backwards compatibility with older Servarr
		SavePath:          savePath,         // Base directory
		ContentPath:       contentPath,      // Full path with torrent ID
		Ratio:             1.0,
		RatioLimit:        -2, // -2 means use global limit
		MaxRatio:          1.0,
		DownloadSpeed:     int64(rdTorrent.Speed),
		TotalSize:         bytesTotal,
	}

	// Parse added time
	if addedOn, err := time.Parse(time.RFC3339Nano, rdTorrent.Added); err == nil {
		info.AddedOn = addedOn.Unix()
	}

	// Set completion info for completed torrents
	if rdTorrent.Status == "downloaded" {
		if endedOn, err := time.Parse(time.RFC3339Nano, rdTorrent.Ended); err == nil {
			info.LastActivity = endedOn.Unix()
			info.CompletionOn = endedOn.Unix()
		}
		info.Availability = 1.0
		info.Completed = bytesTotal
	}

	return info
}

// ConvertTorrentInfoToProperties converts Real-Debrid TorrentInfo to qBittorrent properties
func ConvertTorrentInfoToProperties(rdTorrent *api.TorrentInfo, cfg *config.QBittorrentConfig) TorrentProperties {
	bytesTotal := int64(rdTorrent.Bytes)
	bytesDone := int64(float64(rdTorrent.Bytes) * (rdTorrent.Progress / 100))
	amountLeft := bytesTotal - bytesDone

	eta := int64(8640000)
	if rdTorrent.Status == "downloading" && rdTorrent.Speed > 0 {
		eta = amountLeft / int64(rdTorrent.Speed)
	}

	props := TorrentProperties{
		SavePath:               cfg.SavePath,
		DownloadSpeed:          int64(rdTorrent.Speed),
		DownloadSpeedAvg:       int64(rdTorrent.Speed),
		Eta:                    eta,
		TotalSize:              bytesTotal,
		TotalDownloaded:        bytesDone,
		TotalDownloadedSession: bytesDone,
		ShareRatio:             1.0,
		PieceSize:              16384, // Default piece size
	}

	if addedOn, err := time.Parse(time.RFC3339Nano, rdTorrent.Added); err == nil {
		props.AddedOn = addedOn.Unix()
	}

	if rdTorrent.Status == "downloaded" {
		if endedOn, err := time.Parse(time.RFC3339Nano, rdTorrent.Ended); err == nil {
			props.CompletionOn = endedOn.Unix()
		}
	}

	return props
}

// ConvertRealDebridToProperties converts Real-Debrid torrent to qBittorrent properties
func ConvertRealDebridToProperties(rdTorrent *api.Torrent, cfg *config.QBittorrentConfig) TorrentProperties {
	bytesTotal := int64(rdTorrent.Bytes)
	bytesDone := int64(float64(rdTorrent.Bytes) * (rdTorrent.Progress / 100))
	amountLeft := bytesTotal - bytesDone

	eta := int64(8640000)
	if rdTorrent.Status == "downloading" && rdTorrent.Speed > 0 {
		eta = amountLeft / int64(rdTorrent.Speed)
	}

	props := TorrentProperties{
		SavePath:               cfg.SavePath,
		DownloadSpeed:          int64(rdTorrent.Speed),
		DownloadSpeedAvg:       int64(rdTorrent.Speed),
		Eta:                    eta,
		TotalSize:              bytesTotal,
		TotalDownloaded:        bytesDone,
		TotalDownloadedSession: bytesDone,
		ShareRatio:             1.0,
		PieceSize:              16384, // Default piece size
	}

	if addedOn, err := time.Parse(time.RFC3339Nano, rdTorrent.Added); err == nil {
		props.AddedOn = addedOn.Unix()
	}

	if rdTorrent.Status == "downloaded" {
		if endedOn, err := time.Parse(time.RFC3339Nano, rdTorrent.Ended); err == nil {
			props.CompletionOn = endedOn.Unix()
		}
	}

	return props
}

// ConvertRealDebridFiles converts Real-Debrid files to qBittorrent format
// Note: This function receives TorrentInfo.Files, not just the files array
func ConvertRealDebridFiles(torrentInfo *api.TorrentInfo) []TorrentFile {
	var files []TorrentFile
	for i, rdFile := range torrentInfo.Files {
		// Skip unselected files (matching old implementation)
		if rdFile.Selected == 0 {
			continue
		}

		files = append(files, TorrentFile{
			Index:        i,
			Name:         rdFile.Path,
			Size:         int64(rdFile.Bytes),
			Progress:     torrentInfo.Progress / 100.0, // Use torrent's overall progress
			Priority:     1,                            // Normal priority
			IsSeed:       torrentInfo.Seeders > 0,      // Check if torrent has seeders
			Availability: 1.0,
		})
	}
	return files
}

// ValidatePath checks if a path exists on disk
func ValidatePath(basePath, subPath string) bool {
	fullPath := filepath.Join(basePath, subPath)
	_, err := os.Stat(fullPath)
	return err == nil
}
