package servarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

var apiPath = "/api/v3"

type HistorySinceResponse []Record

type Record struct {
	DownloadID string `json:"downloadId"`
}

func GetHistory(host string, token string) (HistorySinceResponse, error) {
	url, err := url.Parse(host)
	if err != nil {
		return nil, err
	}

	query := url.Query()
	query.Add("eventType", "grabbed")

	url.RawQuery = query.Encode()
	url.Path += apiPath + "/history/since"

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Api-Key", token)

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var data = HistorySinceResponse{}
		if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
			return nil, err
		}

		return data, nil
	case 401:
		return nil, fmt.Errorf("Unauthorized")
	default:
		return nil, fmt.Errorf("Failed to fetch history")
	}
}
