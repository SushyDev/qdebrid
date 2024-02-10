package torrents

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"qdebrid/config"
	"qdebrid/real_debrid"
	"reflect"
	"strings"
	"time"
)

var settings = config.GetSettings()

func Delete(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash := r.FormValue("hashes")

	cachedTorrents, err := getCachedTorrents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var torrent = real_debrid.Torrent{}
	for _, t := range cachedTorrents {
		if t.Hash == hash {
			torrent = t
			break
		}
	}

	if reflect.DeepEqual(torrent, real_debrid.Torrent{}) {
		http.Error(w, "Error fetching torrent", http.StatusInternalServerError)
		return
	}

	if err := real_debrid.Delete(torrent.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Categories(w http.ResponseWriter, r *http.Request) {
	categories := QBitTorrentCategories{
		settings.CategoryName: QBitTorrentCategory{
			Name:     settings.CategoryName,
			SavePath: settings.SavePath,
		},
	}

	jsonData, err := json.Marshal(categories)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func Properties(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")

	cachedTorrents, err := getCachedTorrents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var cachedTorrent = real_debrid.Torrent{}
	for _, torrent := range cachedTorrents {
		if torrent.Hash == hash {
			cachedTorrent = torrent
			break
		}
	}

	if cachedTorrent.Hash != hash {
		http.Error(w, "Cached torrent didn't match hash", http.StatusInternalServerError)
		return
	}

	contentPath := filepath.Join(settings.SavePath, cachedTorrent.Filename)

	addedOn, err := time.Parse(time.RFC3339Nano, cachedTorrent.Added)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	properties := PropertiesResponse{
		AdditionDate: time.Now().Unix(),

		CompletionDate: time.Now().Unix(),
		CreationDate:   addedOn.Unix(),

		DownloadLimit: -1,
		// DownloadSpeed:
		// DownloadSpeedAverage

		LastSeen:  time.Now().Unix(),
		PieceSize: cachedTorrent.Bytes,
		// PiecesHave:
		// PiecesNumber:

		SavePath: contentPath,

		SeedingTime: 1,
		Seeds:       100,
		SeedsTotal:  100,
		ShareRatio:  9999,

		TimeElapsed: time.Now().Unix() - addedOn.Unix(),

		TotalDownloaded:        int64(cachedTorrent.Bytes),
		TotalDownloadedSession: int64(cachedTorrent.Bytes),
		TotalSize:              int64(cachedTorrent.Bytes),
		TotalUploaded:          int64(cachedTorrent.Bytes),
		TotalUploadedSession:   int64(cachedTorrent.Bytes),

		UploadLimit: -1,
		// UploadSpeed:
		// UploadSpeedAverage:
	}

	jsonData, err := json.Marshal(properties)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonData)
}

func Files(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")

	cachedTorrents, err := getCachedTorrents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var id string
	for _, torrent := range cachedTorrents {
		if torrent.Hash == hash {
			id = torrent.ID
			break
		}
	}

	torrent, err := real_debrid.TorrentInfo(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var files = []FileResponse{}
	for index, torrentFile := range torrent.Files {
		file := FileResponse{
			Index:    index,
			Name:     torrentFile.Path,
			Size:     torrentFile.Bytes,
			Progress: 100,
		}

		files = append(files, file)
	}

	jsonData, err := json.Marshal(files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	w.Write(jsonData)
}

func Info(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cachedTorrents, err := getCachedTorrents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userAgent := strings.Split(r.UserAgent(), "/")[0]

	torrentInfos := []TorrentInfo{}
	switch userAgent {
	case "Radarr":
		historyMatches, err := RadarrTorrents(userAgent, cachedTorrents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, match := range historyMatches {
			torrentInfo := GetTorrentInfo(match.Torrent)
			torrentInfos = append(torrentInfos, torrentInfo)
		}

	case "Sonarr":
		historyMatches, err := SonarrTorrents(userAgent, cachedTorrents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, match := range historyMatches {
			torrentInfo := GetTorrentInfo(match.Torrent)
			torrentInfos = append(torrentInfos, torrentInfo)
		}
	default:
		for _, torrent := range cachedTorrents {
			torrentInfo := GetTorrentInfo(torrent)
			torrentInfos = append(torrentInfos, torrentInfo)
		}
	}

	jsonData, err := json.Marshal(torrentInfos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func Add(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	switch r.Header.Get("Content-Type") {
	case "multipart/form-data":
		err := r.ParseMultipartForm(0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Invalid Content-Type", http.StatusInternalServerError)
		return
	}

	urls := r.FormValue("urls")

	for _, url := range SplitString(urls, "\n") {
		if strings.HasPrefix(url, "magnet") {
			if err := real_debrid.AddMagnet(url, "all"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} 

		if strings.HasPrefix(url, "http") {
			torrent, err := GetTorrent(url)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := real_debrid.AddTorrent(torrent, "all"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if r.Header.Get("Content-Type") == "multipart/form-data" {
		torrentHeaders := r.MultipartForm.File["torrents"]

		for _, header := range torrentHeaders {
			torrent, err := header.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := real_debrid.AddTorrent(torrent, "all"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}
