package torrents

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"qdebrid/radarr"
	"qdebrid/real_debrid"
	"qdebrid/sonarr"
	"strings"
	"time"
)

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
		fmt.Println("Failed to fetch history")
		return nil, err
	}

	var sonarrTorrents []SonarrTorrent
	for _, record := range history.Records {
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
		fmt.Println("Failed to fetch history")
		return nil, err
	}

	var radarrTorrents []RadarrTorrent
	for _, record := range history.Records {
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
	addedOn, _ := time.Parse(time.RFC3339Nano, torrent.Added)

	contentPath := filepath.Join(settings.SavePath, torrent.Filename)

	bytesTotal := int64(torrent.Bytes)
	bytesDone := int64(float64(torrent.Bytes) * (torrent.Progress / 100))

	pathExists, _ := PathExists(torrent.Filename)

	state := "checkingUP"
	if !settings.ValidatePaths {
		state = "pausedUP"
	} else if settings.ValidatePaths && pathExists {
		state = "pausedDL"
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
		// DownloadSpeed:

		Downloaded:        bytesDone,
		DownloadedSession: bytesDone,

		Hash: torrent.Hash,

		LastActivity: time.Now().Unix(),

		MaxRatio:       -1,
		MaxSeedingTime: -1,

		Name: torrent.Hash,

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

		// use fs to check if direcotry exists and set state between pending and pausedUP
		State: state,

		TimeActive: time.Now().Unix() - addedOn.Unix(),

		TotalSize: bytesTotal,

		Tracker: "udp://tracker.opentrackr.org:1337",

		UploadLimit:     -1,
		Uploaded:        bytesDone,
		UploadedSession: bytesDone,
		// UploadSpeed:
	}
}
