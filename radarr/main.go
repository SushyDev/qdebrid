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

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return HistorySinceResponse{}, fmt.Errorf("Failed to fetch history")
	}

	var data = HistorySinceResponse{}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	return data, nil
}
