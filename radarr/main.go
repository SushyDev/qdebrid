package radarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"qdebrid/config"
)

var apiPath = "/api/v3"

var settings = config.GetSettings()

func History() (HistorySinceResponse, error) {
	url, err := url.Parse(settings.Radarr.Host)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	url.Path += apiPath + "/history/since"
	url.Query().Add("date", "1970-01-01")
	url.Query().Add("eventType", "1")

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	req.Header.Set("X-Api-Key", settings.Radarr.Token)

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	switch response.StatusCode {
	case 200:
		var data = HistorySinceResponse{}
		if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
			return HistorySinceResponse{}, err
		}

		return data, nil
	case 401:
		return HistorySinceResponse{}, fmt.Errorf("Unauthorized")
	default:
		return HistorySinceResponse{}, fmt.Errorf("Failed to fetch history")
	}
}
