package torrents

import (
	"encoding/json"
	"fmt"
	"net/http"
	"qdebrid/config"
	"qdebrid/logger"
	"qdebrid/real_debrid"
	"qdebrid/servarr"
	"strings"
)

var settings = config.GetSettings()

var sugar = logger.Sugar()

func Delete(w http.ResponseWriter, r *http.Request) {
	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/delete", "Received request to delete torrent(s)"))

	err := r.ParseForm()
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/delete", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hash := r.FormValue("hashes")

	torrent, err := FindCachedTorrent(hash)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/delete", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := real_debrid.Delete(torrent.ID); err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/delete", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Categories(w http.ResponseWriter, r *http.Request) {
	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/categories", "Received request for torrent categories"))
	
	categories := map[string]Category{
		settings.QDebrid.CategoryName: {
			Name:     settings.QDebrid.CategoryName,
			SavePath: settings.QDebrid.SavePath,
		},
	}

	jsonData, err := json.Marshal(categories)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/categories", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sugar.Debug(logger.EndpointMessage("qbittorrent", "torrents/categories", fmt.Sprintf("Categories: %s", jsonData)))

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func Properties(w http.ResponseWriter, r *http.Request) {
	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/properties", "Received request for torrent properties"))

	hash := r.URL.Query().Get("hash")

	torrent, err := FindCachedTorrent(hash)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/delete", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	torrentInfo, err := GetTorrentInfo(torrent)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/properties", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonData, err := json.Marshal(torrentInfo)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/properties", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func Files(w http.ResponseWriter, r *http.Request) {
	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/files", "Received request for torrent files"))

	hash := r.URL.Query().Get("hash")

	torrent, err := FindCachedTorrent(hash)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/delete", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	torrentInfo, err := real_debrid.TorrentInfo(torrent.ID)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/files", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var files = []FileResponse{}
	for index, torrentFile := range torrentInfo.Files {
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
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/files", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func Info(w http.ResponseWriter, r *http.Request) {
	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/info", "Received request for torrent info"))

	cachedTorrents, err := getCachedTorrents()
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/info", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	host, token, err := DecodeAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/info", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	torrentInfos := []TorrentInfo{}
	history, err := servarr.History(host, token)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/info", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	historyMatches, err := ServarrTorrents(history, cachedTorrents)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/info", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, match := range historyMatches {
		torrentInfo, err := GetTorrentInfo(match.Torrent)
		if err != nil {
			sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/info", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		torrentInfos = append(torrentInfos, torrentInfo)
	}

	jsonData, err := json.Marshal(torrentInfos)
	if err != nil {
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/info", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sugar.Debug(logger.EndpointMessage("qbittorrent", "torrents/info", fmt.Sprintf("Returned %d matches", len(torrentInfos))))

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func Add(w http.ResponseWriter, r *http.Request) {
	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/add", "Received request to add torrent(s)"))

	contentType := strings.Split(r.Header.Get("Content-Type"), ";")[0]

	sugar.Debug(logger.EndpointMessage("qbittorrent", "torrents/add", fmt.Sprintf("Content-Type: %s", contentType)))

	switch contentType {
	case "multipart/form-data":
		err := r.ParseMultipartForm(0)
		if err != nil {
			sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "application/x-www-form-urlencoded":
		err := r.ParseForm()
		if err != nil {
			sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", "Invalid Content-Type"))
		http.Error(w, "Invalid Content-Type", http.StatusInternalServerError)
		return
	}

	added := 0

	urls := r.FormValue("urls")
	for _, url := range SplitString(urls, "\n") {
		if strings.HasPrefix(url, "magnet") {
			if err := real_debrid.AddMagnet(url); err != nil {
				sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			added++
		}

		if strings.HasPrefix(url, "http") {
			torrent, err := GetTorrent(url)
			if err != nil {
				sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := real_debrid.AddTorrent(torrent); err != nil {
				sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			added++
		}
	}

	if contentType == "multipart/form-data" {
		torrentHeaders := r.MultipartForm.File["torrents"]

		for _, header := range torrentHeaders {
			torrent, err := header.Open()
			if err != nil {
				sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := real_debrid.AddTorrent(torrent); err != nil {
				sugar.Error(logger.EndpointMessage("qbittorrent", "torrents/add", err.Error()))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			added++
		}
	}

	sugar.Info(logger.EndpointMessage("qbittorrent", "torrents/add", fmt.Sprintf("Added %d torrents", added)))

	ClearCachedTorrents()

	w.WriteHeader(http.StatusOK)
}
