package real_debrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func Torrents() (TorrentsResponse, error) {
	url, _ := url.Parse(apiHost)

	query := url.Query()
	query.Add("limit", "1000")
	query.Add("page", "1")

	url.RawQuery = query.Encode()
	url.Path += apiPath + "/torrents"

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return TorrentsResponse{}, err
	}

	response, err := client.Do(req)
	if err != nil {
		return TorrentsResponse{}, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var torrents = TorrentsResponse{}
		if err := json.NewDecoder(response.Body).Decode(&torrents); err != nil {
			return TorrentsResponse{}, err
		}

		return torrents, nil
	case 401:
		return TorrentsResponse{}, fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return TorrentsResponse{}, fmt.Errorf("Permission denied (account locked, not premium)")
	default:
		return TorrentsResponse{}, fmt.Errorf("Unknown error")
	}
}
