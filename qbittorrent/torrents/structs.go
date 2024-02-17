package torrents

import (
	"qdebrid/real_debrid"
	"qdebrid/servarr"
)

type Category struct {
	Name     string `json:"name"`
	SavePath string `json:"savePath"`
}

type PropertiesResponse struct {
	SavePath               string  `json:"save_path"`
	CreationDate           int64   `json:"creation_date"`
	PieceSize              int     `json:"piece_size"`
	Comment                string  `json:"comment"`
	TotalWasted            int64   `json:"total_wasted"`
	TotalUploaded          int64   `json:"total_uploaded"`
	TotalUploadedSession   int64   `json:"total_uploaded_session"`
	TotalDownloaded        int64   `json:"total_downloaded"`
	TotalDownloadedSession int64   `json:"total_downloaded_session"`
	UploadLimit            int     `json:"up_limit"`
	DownloadLimit          int     `json:"dl_limit"`
	TimeElapsed            int64   `json:"time_elapsed"`
	SeedingTime            int     `json:"seeding_time"`
	NbConnections          int     `json:"nb_connections"`
	NbConnectionsLimit     int     `json:"nb_connections_limit"`
	ShareRatio             float64 `json:"share_ratio"`
	AdditionDate           int64   `json:"addition_date"`
	CompletionDate         int64   `json:"completion_date"`
	CreatedBy              string  `json:"created_by"`
	DownloadSpeedAverage   int64   `json:"dl_speed_avg"`
	DownloadSpeed          int64   `json:"dl_speed"`
	ETA                    int64   `json:"eta"`
	LastSeen               int64   `json:"last_seen"`
	Peers                  int     `json:"peers"`
	PeersTotal             int     `json:"peers_total"`
	PiecesHave             int     `json:"pieces_have"`
	PiecesNumber           int     `json:"pieces_num"`
	Reannounce             int     `json:"reannounce"`
	Seeds                  int     `json:"seeds"`
	SeedsTotal             int     `json:"seeds_total"`
	TotalSize              int64   `json:"total_size"`
	UploadSpeedAverage     int     `json:"up_speed_avg"`
	UploadSpeed            int     `json:"up_speed"`
}

type FileResponse struct {
	Index        int     `json:"index"`
	Name         string  `json:"name"`
	Size         int     `json:"size"`
	Progress     float64 `json:"progress"`
	Priority     int     `json:"priority"`
	IsSeed       bool    `json:"is_seed"`
	PieceRange   [2]int  `json:"piece_range"`
	Availability float64 `json:"availability"`
}

type Torrent struct {
	DLSpeed       int     `json:"dlspeed"`
	ETA           int     `json:"eta"`
	FLPiecePrio   bool    `json:"f_l_piece_prio"`
	ForceStart    bool    `json:"force_start"`
	Hash          string  `json:"hash"`
	Category      string  `json:"category"`
	Tags          string  `json:"tags"`
	Name          string  `json:"name"`
	NumComplete   int     `json:"num_complete"`
	NumIncomplete int     `json:"num_incomplete"`
	NumLeechs     int     `json:"num_leechs"`
	NumSeeds      int     `json:"num_seeds"`
	Priority      int     `json:"priority"`
	Progress      float64 `json:"progress"`
	Ratio         float64 `json:"ratio"`
	SeqDL         bool    `json:"seq_dl"`
	Size          int     `json:"size"`
	State         string  `json:"state"`
	SuperSeeding  bool    `json:"super_seeding"`
	UPSpeed       int     `json:"upspeed"`
}

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

type ServarrTorrentMatch struct {
	History servarr.Record
	Torrent real_debrid.Torrent
}
