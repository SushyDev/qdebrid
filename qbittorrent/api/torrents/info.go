package torrents

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"qdebrid/qbittorrent/helpers"
	"strings"

	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func decodeAuthHeader(request *http.Request) (string, string, error) {
	header := request.Header.Get("Authorization")
	if header == "" {
		return "", "", fmt.Errorf("Authorization header is missing")
	}

	encodedToken := strings.Split(header, " ")[1]

	bytes, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return "", "", err
	}

	bearer := string(bytes)

	colonIndex := strings.LastIndex(bearer, ":")
	host := bearer[:colonIndex]
	token := bearer[colonIndex+1:]

	return host, token, nil
}

func (module *Module) Info(w http.ResponseWriter, request *http.Request) {
	logger := module.GetLogger()

	logger.Info("Received request for torrent info")

	torrents, err := real_debrid_api.GetTorrents(module.RealDebridClient)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	torrentInfos := []helpers.TorrentInfo{}

	for _, match := range *torrents {
		torrentInfo, err := helpers.GetTorrentInfo(match)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		torrentInfos = append(torrentInfos, torrentInfo)
	}

	jsonData, err := json.Marshal(torrentInfos)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Debug(fmt.Sprintf("Returned %d matches", len(torrentInfos)))

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
