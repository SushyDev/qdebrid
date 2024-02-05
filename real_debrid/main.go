package real_debrid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"qdebrid/config"
)

type AddMagnetResponse struct {
	Id  string `json:"id"`
	Uri string `json:"uri"`
}

var apiHost = "https://api.real-debrid.com"
var apiPath = "/rest/1.0"

var settings = config.GetSettings()

func InstantAvailability(hash string) (InstantAvailabilityResponse, error) {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/instantAvailability/" + hash
	
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Println("Failed to create request")
		return InstantAvailabilityResponse{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
		return InstantAvailabilityResponse{}, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var instantAvailability = InstantAvailabilityResponse{}
		err = json.NewDecoder(response.Body).Decode(&instantAvailability)
		if err != nil {
			fmt.Println("Failed to decode response")
			return InstantAvailabilityResponse{}, err
		}

		return instantAvailability, nil
	case 401:
		return InstantAvailabilityResponse{}, fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return InstantAvailabilityResponse{}, fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return InstantAvailabilityResponse{}, fmt.Errorf("Unknown ressource (invalid id)")
	default:
		return InstantAvailabilityResponse{}, fmt.Errorf("Unknown error")
	}
}


func Torrents() (TorrentsResponse, error) {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents"

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Println("Failed to create request")
		return TorrentsResponse{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
		return TorrentsResponse{}, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var torrents = TorrentsResponse{}
		err = json.NewDecoder(response.Body).Decode(&torrents)
		if err != nil {
			fmt.Println("Failed to decode response")
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

func TorrentInfo(infoHash string) (TorrentInfoResponse, error) {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/info/" + infoHash

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		fmt.Println("Failed to create request")
		return TorrentInfoResponse{}, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
		return TorrentInfoResponse{}, err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		var torrentInfo = TorrentInfoResponse{}
		err = json.NewDecoder(response.Body).Decode(&torrentInfo)
		if err != nil {
			fmt.Println("Failed to decode response")
			return TorrentInfoResponse{}, err
		}

		return torrentInfo, nil
	case 401:
		return TorrentInfoResponse{}, fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return TorrentInfoResponse{}, fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return TorrentInfoResponse{}, fmt.Errorf("Unknown ressource (invalid id)")
	default:
		return TorrentInfoResponse{}, fmt.Errorf("Unknown error")
	}
}

func AddMagnet(magnet string, files string) error {
	input := url.Values{}
	input.Set("magnet", magnet)
	requestBody := input.Encode()

	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/addMagnet"

	req, err := http.NewRequest("POST", url.String(), bytes.NewBufferString(requestBody))
	if err != nil {
		fmt.Println("Failed to create request")
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
		return err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 201:
		var data AddMagnetResponse
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			fmt.Println("Failed to decode response")
			return err
		}

		return selectFiles(data.Id, files)
	case 400:
		return fmt.Errorf("Bad Request (see error message)")
	case 401:
		return fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return fmt.Errorf("Permission denied (account locked, not premium) or Infringing torrent")
	case 503:
		return fmt.Errorf("Service unavailable (see error message)")
	case 504:
		return fmt.Errorf("Service timeout (see error message)")
	default:
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Failed to read response body")
			return err
		}

		fmt.Println(string(body))

		return fmt.Errorf("[%v] Unknown error", response.StatusCode)
	}
}

func AddTorrent(torrent multipart.File, files string) error {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/addTorrent"

	req, err := http.NewRequest("PUT", url.String(), torrent)
	if err != nil {
		fmt.Println("Failed to create request")
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
		return err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 201:
		var data AddMagnetResponse
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			fmt.Println("Failed to decode response")
			return err
		}

		return selectFiles(data.Id, files)
	case 400:
		return fmt.Errorf("Bad Request (see error message)")
	case 401:
		return fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return fmt.Errorf("Permission denied (account locked, not premium) or Infringing torrent")
	case 503:
		return fmt.Errorf("Service unavailable (see error message)")
	case 504:
		return fmt.Errorf("Service timeout (see error message)")
	default:
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Failed to read response")
			return err
		}

		fmt.Println(string(body))

		return fmt.Errorf("[%v] Unknown error", response.StatusCode)
	}
}


func Delete(id string) error {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/delete/" + id

	req, err := http.NewRequest("DELETE", url.String(), nil)
	if err != nil {
		fmt.Println("Failed to create request")
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
		return err
	}

	defer response.Body.Close()

	switch response.StatusCode {
	case 204:
		return nil
	case 401:
		return fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return fmt.Errorf("Unknown ressource (invalid id)")
	default:
		return fmt.Errorf("Unknown error")
	}
}

func selectFiles(id string, files string) error {
	input := url.Values{}
	input.Set("files", files)

	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/selectFiles/" + id

	requestBody := input.Encode()
	req, err := http.NewRequest("POST", url.String(), bytes.NewBufferString(requestBody))
	if err != nil {
		fmt.Println("Failed to create request")
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.RealDebrid.Token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to send request")
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

		return fmt.Errorf("Wrong parameter (invalid file id(s)) / Unknown ressource (invalid id)")
	default:
		return fmt.Errorf("Unknown error")
	}
}
