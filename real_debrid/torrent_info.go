package real_debrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func TorrentInfo(id string) (TorrentInfoResponse, error) {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/info/" + id

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return TorrentInfoResponse{}, err
	}

	response, err := client.Do(req)
	if err != nil {
		return TorrentInfoResponse{}, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var torrentInfo = TorrentInfoResponse{}
		err = json.NewDecoder(response.Body).Decode(&torrentInfo)
		if err != nil {
			return TorrentInfoResponse{}, err
		}

		return torrentInfo, nil
	case 401:
		return TorrentInfoResponse{}, fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return TorrentInfoResponse{}, fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return TorrentInfoResponse{}, fmt.Errorf("Unknown resource (invalid id)")
	default:
		return TorrentInfoResponse{}, fmt.Errorf("Unknown error")
	}
}
