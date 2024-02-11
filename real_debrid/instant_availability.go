package real_debrid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func instantAvailability(hash string) (InstantAvailabilityResponse, error) {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/instantAvailability/" + hash

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return InstantAvailabilityResponse{}, err
	}

	response, err := client.Do(req)
	if err != nil {
		return InstantAvailabilityResponse{}, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var instantAvailability = InstantAvailabilityResponse{}
		if err := json.NewDecoder(response.Body).Decode(&instantAvailability); err != nil {
			return InstantAvailabilityResponse{}, err
		}

		return instantAvailability, nil
	case 401:
		return InstantAvailabilityResponse{}, fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return InstantAvailabilityResponse{}, fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return InstantAvailabilityResponse{}, fmt.Errorf("Unknown resource (invalid id): %s", hash)
	default:
		return InstantAvailabilityResponse{}, fmt.Errorf("Unknown error")
	}
}
