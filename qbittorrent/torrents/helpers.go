package torrents

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"qdebrid/real_debrid"
	"qdebrid/servarr"
	"reflect"
	"strings"
	"time"
)

var _cachedTorrents = real_debrid.TorrentsResponse{}
var _cachedTorrentsTime = time.Now()

func DecodeAuthHeader(header string) (string, string, error) {
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

func getCachedTorrents() (real_debrid.TorrentsResponse, error) {
	passed := time.Now().Sub(_cachedTorrentsTime)

	cacheTTL, err := time.ParseDuration(settings.QDebrid.ResponseCacheTTL)
	if err != nil {
		return real_debrid.TorrentsResponse{}, err
	}

	cacheInvalid := passed > cacheTTL

	if !reflect.DeepEqual(_cachedTorrents, real_debrid.TorrentsResponse{}) && !cacheInvalid {
		return _cachedTorrents, nil
	}

	torrents, err := real_debrid.Torrents()
	if err != nil {
		return real_debrid.TorrentsResponse{}, err
	}

	_cachedTorrents = torrents
	_cachedTorrentsTime = time.Now()

	return _cachedTorrents, nil
}

func ClearCachedTorrents() {
	sugar.Debug("Cleared cached torrents")

	_cachedTorrents = real_debrid.TorrentsResponse{}
}

func FindCachedTorrent(hash string) (real_debrid.Torrent, error) {
	cachedTorrents, err := getCachedTorrents()
	if err != nil {
		return real_debrid.Torrent{}, err
	}

	var torrent = real_debrid.Torrent{}
	for _, t := range cachedTorrents {
		if t.Hash == hash {
			torrent = t
			break
		}
	}

	if reflect.DeepEqual(torrent, real_debrid.Torrent{}) {
		return real_debrid.Torrent{}, fmt.Errorf("Couldn't find hash in cached torrents")
	}

	return torrent, nil
}

func SplitString(s string, sep string) []string {
	reader := strings.NewReader(s)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var result []string
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return result
}

func ServarrTorrents(history servarr.HistorySinceResponse, torrents []real_debrid.Torrent) ([]ServarrTorrentMatch, error) {
	var servarrTorrents []ServarrTorrentMatch
	for _, record := range history {
	torrents:
		for _, torrent := range torrents {
			if strings.EqualFold(record.DownloadID, torrent.Hash) {
				for _, existing := range servarrTorrents {
					if existing.Torrent.Hash == torrent.Hash {
						break torrents
					}
				}

				servarrTorrent := ServarrTorrentMatch{
					History: record,
					Torrent: torrent,
				}

				servarrTorrents = append(servarrTorrents, servarrTorrent)
			}
		}
	}

	return servarrTorrents, nil
}

func GetTorrent(url string) (io.Reader, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, err
	}

	return response.Body, nil
}

func PathExists(path string) (bool, error) {
	url, _ := url.Parse(settings.Zurg.Host)
	url.Path += "/http/__all__/" + path + "/"

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode == 200 {
		return true, nil
	}

	return false, nil
}

func GetTorrentInfo(torrent real_debrid.Torrent) (TorrentInfo, error) {
	var state string
	switch torrent.Status {
	case "magnet_error":
		state = "error"
	case "magnet_conversion":
		state = "checkingUP"
	case "waiting_files_selection":
		state = "checkingUP"
	case "queued":
		state = "checkingUP"
	case "downloading":
		state = "downloading"
	case "downloaded":
		state = "pausedUP"
	case "error":
		state = "error"
	case "virus":
		state = "error"
	case "compressing":
		state = "checkingUP"
	case "uploading":
		state = "uploading"
	case "dead":
		state = "error"
	default:
		state = "unknown"
	}

	pathExists, err := PathExists(torrent.Filename)
	if err != nil {
		return TorrentInfo{}, err
	}

	if state == "pausedUP" && settings.QDebrid.ValidatePaths && !pathExists {
		state = "checkingUP"
	}

	addedOn, err := time.Parse(time.RFC3339Nano, torrent.Added)
	if err != nil {
		return TorrentInfo{}, err
	}

	contentPath := filepath.Join(settings.QDebrid.SavePath, torrent.Filename)

	bytesTotal := int64(torrent.Bytes)
	bytesDone := int64(float64(torrent.Bytes) * (torrent.Progress / 100))

	var speed int64
	if torrent.Speed != 0 {
		speed = int64(torrent.Speed)
	}

	var eta int64
	if speed != 0 {
		eta = (bytesTotal - bytesDone) / speed
	}

	return TorrentInfo{
		AddedOn:    addedOn.Unix(),
		AmountLeft: bytesTotal - bytesDone,

		// Availability: 2,

		Category: settings.QDebrid.CategoryName,

		Completed:    bytesDone,
		CompletionOn: addedOn.Unix(),

		ContentPath: contentPath,

		// DownloadLimit: -1,
		DownloadSpeed: speed,

		ETA: eta,

		Downloaded:        bytesDone,
		DownloadedSession: bytesDone,

		Hash: torrent.Hash,

		LastActivity: time.Now().Unix(),

		// MaxRatio:       -1,
		// MaxSeedingTime: -1,

		Name: torrent.Filename,

		// NumComplete:   10,
		// NumIncomplete: 0,
		// NumLeechs:     100,
		// NumSeeds:      100,

		// Priority: 999,

		Progress: torrent.Progress / 100,

		// Ratio:      1,
		// RatioLimit: 1,

		SavePath: contentPath,

		// SeedingTimeLimit: 1,

		SeenComplete: time.Now().Unix(),

		Size: bytesTotal,

		State: state,

		TimeActive: time.Now().Unix() - addedOn.Unix(),

		TotalSize: bytesTotal,

		// Tracker: "udp://tracker.opentrackr.org:1337",

		// UploadLimit:     -1,
		// Uploaded:        bytesDone,
		// UploadedSession: bytesDone,
	}, nil
}
