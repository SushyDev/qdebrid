package torrents

import (
	"bufio"
	"net/http"
	"net/url"
	"path/filepath"
	"qdebrid/radarr"
	"qdebrid/real_debrid"
	"qdebrid/sonarr"
	"reflect"
	"strings"
	"time"
)

// ZCACHE
var _cachedTorrents = real_debrid.TorrentsResponse{}
var _cachedTorrentsTime = time.Now()

// Return cache if date is < 5 minutes
func getCachedTorrents() (real_debrid.TorrentsResponse, error) {
	cacheInvalid := time.Now().Sub(_cachedTorrentsTime) > 5 * time.Minute

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

type SonarrTorrent struct {
	History sonarr.Record
	Torrent real_debrid.Torrent
}

type RadarrTorrent struct {
	History radarr.Record
	Torrent real_debrid.Torrent
}

func SonarrTorrents(userAgent string, torrents []real_debrid.Torrent) ([]SonarrTorrent, error) {
	history, err := sonarr.History()
	if err != nil {
		return nil, err
	}

	var sonarrTorrents []SonarrTorrent
	for _, record := range history {
	torrents:
		for _, torrent := range torrents {
			if strings.EqualFold(record.DownloadID, torrent.Hash) {
				for _, existing := range sonarrTorrents {
					if existing.Torrent.Hash == torrent.Hash {
						break torrents
					}
				}

				sonarrTorrent := SonarrTorrent{
					History: record,
					Torrent: torrent,
				}

				sonarrTorrents = append(sonarrTorrents, sonarrTorrent)
			}
		}
	}

	return sonarrTorrents, nil
}

func RadarrTorrents(userAgent string, torrents []real_debrid.Torrent) ([]RadarrTorrent, error) {
	history, err := radarr.History()
	if err != nil {
		return nil, err
	}

	var radarrTorrents []RadarrTorrent
	for _, record := range history {
	torrents:
		for _, torrent := range torrents {
			if strings.EqualFold(record.DownloadID, torrent.Hash) {
				for _, existing := range radarrTorrents {
					if existing.Torrent.Hash == torrent.Hash {
						break torrents
					}
				}

				radarrTorrent := RadarrTorrent{
					History: record,
					Torrent: torrent,
				}

				radarrTorrents = append(radarrTorrents, radarrTorrent)
			}
		}
	}

	return radarrTorrents, nil
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

func GetTorrentInfo(torrent real_debrid.Torrent) TorrentInfo {
	var state string
	switch torrent.Status {
	case "magnet_error":
		state = "error"
	case "magnet_conversion":
		state = "checkingUP"
	case "waiting_files_selection":
		state = "pausedUP"
	case "queued":
		state = "pausedUP"
	case "downloading":
		state = "downloading"
	case "downloaded":
		state = "pausedUP"
	case "error":
		state = "error"
	case "virus":
		state = "error"
	case "compressing":
		state = "pausedUP"
	case "uploading":
		state = "uploading"
	case "dead":
		state = "error"
	default:
		state = "checkingUP"
	}

	pathExists, _ := PathExists(torrent.Filename)
	if state == "pausedUP" {
		if !settings.ValidatePaths {
		} else if settings.ValidatePaths && pathExists {
			state = "pausedUP"
		}
	}

	addedOn, _ := time.Parse(time.RFC3339Nano, torrent.Added)

	contentPath := filepath.Join(settings.SavePath, torrent.Filename)

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

		Availability: 2,

		Category: settings.CategoryName,

		Completed:    bytesDone,
		CompletionOn: addedOn.Unix(),

		ContentPath: contentPath,

		DownloadLimit: -1,
		DownloadSpeed: speed,

		ETA: eta,

		Downloaded:        bytesDone,
		DownloadedSession: bytesDone,

		Hash: torrent.Hash,

		LastActivity: time.Now().Unix(),

		MaxRatio:       -1,
		MaxSeedingTime: -1,

		Name: torrent.Filename,

		NumComplete:   10,
		NumIncomplete: 0,
		NumLeechs:     100,
		NumSeeds:      100,

		Priority: 999,

		Progress: torrent.Progress / 100,

		Ratio:      1,
		RatioLimit: 1,

		SavePath: contentPath,

		SeedingTimeLimit: 1,

		SeenComplete: time.Now().Unix(),

		Size: bytesTotal,

		State: state,

		TimeActive: time.Now().Unix() - addedOn.Unix(),

		TotalSize: bytesTotal,

		Tracker: "udp://tracker.opentrackr.org:1337",

		UploadLimit:     -1,
		Uploaded:        bytesDone,
		UploadedSession: bytesDone,
	}
}
