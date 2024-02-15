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

func History() (HistorySinceResponse, error) {
	url, err := url.Parse(settings.Sonarr.Host)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	query := url.Query()
	query.Add("eventType", "grabbed")

	url.RawQuery = query.Encode()
	url.Path += apiPath + "/history/since"

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	req.Header.Set("X-Api-Key", settings.Sonarr.Token)

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return HistorySinceResponse{}, err
	}

	defer response.Body.Close()

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
