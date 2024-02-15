package real_debrid

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func filterFiles(torrent TorrentInfoResponse) ([]string, error) {
	// In the future might want to throw an error and make AllowedFileTypes required
	if len(settings.QDebrid.AllowedFileTypes) == 0 {
		return []string{"all"}, nil
	}

	var ids []string
	for _, file := range torrent.Files {
		if file.Bytes <= settings.QDebrid.MinFileSize {
			continue
		}

		for _, extension := range settings.QDebrid.AllowedFileTypes {
			if strings.HasSuffix(file.Path, extension) {
				ids = append(ids, strconv.Itoa(file.ID))
			}
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("No accepted files found")
	}

	return ids, nil
}

func filesAvailability(hash string, ids []string) error {
	if settings.QDebrid.AllowUncached {
		return nil
	}

	available, err := instantAvailability(hash)
	if err != nil {
		return err
	}

	idsMap := make(map[string]bool)
	for _, id := range ids {
		idsMap[id] = true
	}

variants:
	for _, variant := range available[hash]["rd"] {
		if len(variant) != len(ids) {
			continue
		}

		for ids := range variant {
			if !idsMap[ids] {
				continue variants
			}
		}

		return nil
	}

	return fmt.Errorf("No cached files available")
}

func selectFiles(id string) error {
	torrent, err := TorrentInfo(id)
	if err != nil {
		return DeleteBecauseError(id, err)
	}

	files, err := filterFiles(torrent)
	if err != nil {
		return DeleteBecauseError(id, err)
	}

	err = filesAvailability(torrent.Hash, files)
	if err != nil {
		return DeleteBecauseError(id, err)
	}

	var input = url.Values{}
	input.Set("files", strings.Join(files, ","))
	requestBody := input.Encode()

	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/selectFiles/" + id

	req, err := http.NewRequest("POST", url.String(), bytes.NewBufferString(requestBody))
	if err != nil {
		return DeleteBecauseError(id, err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(req)
	if err != nil {
		return DeleteBecauseError(id, err)
	}

	defer response.Body.Close()

	err = nil
	switch response.StatusCode {
	case 202:
		err = fmt.Errorf("Action already done")
	case 204:
		err = nil
	case 400:
		err = fmt.Errorf("Bad Request (see error message)")
	case 401:
		err = fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		err = fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		err = fmt.Errorf("Wrong parameter (invalid file id(s)) / Unknown resource (invalid id)")
	default:
		err = fmt.Errorf("Unknown error")
	}

	if err != nil {
		return DeleteBecauseError(id, err)
	}

	return nil
}
