package helpers

import (
	"path/filepath"
	"time"

	"github.com/sushydev/real_debrid_go/api"
)

type TorrentProperties struct {
	SavePath               string    `json:"save_path"`
	CreationDate           time.Time `json:"creation_date"`
	PieceSize              int       `json:"piece_size"`
	Comment                string    `json:"comment"`
	TotalWasted            int       `json:"total_wasted"`
	TotalUploaded          int       `json:"total_uploaded"`
	TotalUploadedSession   int       `json:"total_uploaded_session"`
	TotalDownloaded        int       `json:"total_downloaded"`
	TotalDownloadedSession int       `json:"total_downloaded_session"`
	UpLimit                int       `json:"up_limit"`
	DlLimit                int       `json:"dl_limit"`
	TimeElapsed            int       `json:"time_elapsed"`
	SeedingTime            int       `json:"seeding_time"`
	NbConnections          int       `json:"nb_connections"`
	NbConnectionsLimit     int       `json:"nb_connections_limit"`
	ShareRatio             float64   `json:"share_ratio"`
	AdditionDate           time.Time `json:"addition_date"`
	CompletionDate         time.Time `json:"completion_date"`
	CreatedBy              string    `json:"created_by"`
	DlSpeedAvg             int       `json:"dl_speed_avg"`
	DlSpeed                int       `json:"dl_speed"`
	ETA                    int       `json:"eta"`
	LastSeen               time.Time `json:"last_seen"`
	Peers                  int       `json:"peers"`
	PeersTotal             int       `json:"peers_total"`
	PiecesHave             int       `json:"pieces_have"`
	PiecesNum              int       `json:"pieces_num"`
	Reannounce             int       `json:"reannounce"`
	Seeds                  int       `json:"seeds"`
	SeedsTotal             int       `json:"seeds_total"`
	TotalSize              int       `json:"total_size"`
	UpSpeedAvg             int       `json:"up_speed_avg"`
	UpSpeed                int       `json:"up_speed"`
	IsPrivate              bool      `json:"is_private"`
}

func GetTorrentProperties(torrent *api.TorrentInfo) (TorrentProperties, error) {
	addedOn, err := time.Parse(time.RFC3339Nano, torrent.Added)
	if err != nil {
		return TorrentProperties{}, err
	}

	contentPath := filepath.Join(settings.QDebrid.SavePath, torrent.ID)
	bytesTotal := torrent.Bytes
	bytesDone := torrent.Bytes * int((torrent.Progress / 100))
	eta := 60 * 60 * 24 * 365

	torrentProperties := TorrentProperties{
		SavePath:     contentPath,
		CreationDate: addedOn,
		AdditionDate: addedOn,
		ETA:          eta,
		Seeds:        torrent.Seeders,
	}

	if torrent.Status == "downloading" {
		if torrent.Speed < 0 {
			torrentProperties.ETA = int((bytesTotal - bytesDone) / torrent.Speed)
		}
	}

	if torrent.Status == "downloaded" {
		endedOn, err := time.Parse(time.RFC3339Nano, torrent.Ended)
		if err != nil {
			return TorrentProperties{}, err
		}

		torrentProperties.CompletionDate = endedOn
		torrentProperties.TotalDownloaded = bytesTotal
		torrentProperties.LastSeen = endedOn

	}

	return torrentProperties, nil
}
