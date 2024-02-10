package real_debrid

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func filesToSelect(hash string) ([]string, error) {
	available, err := instantAvailability(hash)
	if err != nil {
		return nil, err
	}

	var ids []string

	variantLoop:
	for _, variant := range available[hash]["rd"] {
		if len(settings.QDebrid.AllowedFileTypes) == 0 {
			for id, _ := range variant {
				ids = append(ids, id)
			}
		}

		for _, extension := range settings.QDebrid.AllowedFileTypes {
			for _, file := range variant {
				if strings.HasSuffix(file.FileName, extension) {
					for id, _ := range variant {
						ids = append(ids, id)
					}

					continue variantLoop
				}
			}
		}
	}

	if !settings.QDebrid.AllowUncached && len(ids) == 0 {
		return nil, fmt.Errorf("No cached files available")
	}

	return ids, nil
}

func selectFiles(id string) error {
	torrent, err := TorrentInfo(id)
	if err != nil {
		return err
	}

	files, err := filesToSelect(torrent.Hash)
	if err != nil {
		return err
	}

	var input = url.Values{}
	input.Set("files", strings.Join(files, ","))
	requestBody := input.Encode()

	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/selectFiles/" + id

	req, err := http.NewRequest("POST", url.String(), bytes.NewBufferString(requestBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 202:
		return fmt.Errorf("Action already done")
	case 204:
		return nil
	case 400:
		return fmt.Errorf("Bad Request (see error message)")
	case 401:
		return fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		err := Delete(id)
		if err != nil {
			return err
		}

		return fmt.Errorf("Wrong parameter (invalid file id(s)) / Unknown resource (invalid id)")
	default:
		return fmt.Errorf("Unknown error")
	}
}
