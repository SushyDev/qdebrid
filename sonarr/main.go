package sonarr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"qdebrid/config"
)

var apiPath = "/api/v3"

var settings = config.GetSettings()

func History() (HistoryResponse, error) {
	url, err := url.Parse(settings.Sonarr.Host)
	if err != nil {
		return HistoryResponse{}, err
	}

	url.Path += apiPath + "/history"
	url.Query().Add("eventType", "1")
	url.Query().Add("pageSize", "250") // TODO

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return HistoryResponse{}, err
	}

	req.Header.Set("X-Api-Key", settings.Sonarr.Token)

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return HistoryResponse{}, err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return HistoryResponse{}, fmt.Errorf("Failed to fetch history")
	}

	var data = HistoryResponse{}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return HistoryResponse{}, err
	}

	return data, nil
}
