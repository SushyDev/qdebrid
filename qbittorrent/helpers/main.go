package helpers

import (
	"net/http"
	"net/url"
	"qdebrid/config"
)

var settings = config.GetSettings()

func pathExists(path string) (bool, error) {
	url, _ := url.Parse("/") // todo zurg something blah
	url.Path += "/http/__all__/" + path + "/"

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode == 200 {
		return true, nil
	}

	return false, nil
}

func mapRealDebridStatus(status string) string {
	switch status {
	case "magnet_error":
		return "error"
	case "magnet_conversion":
		return "checkingUP"
	case "waiting_files_selection":
		return "checkingUP"
	case "queued":
		return "checkingUP"
	case "downloading":
		return "downloading"
	case "downloaded":
		return "pausedUP"
	case "error":
		return "error"
	case "virus":
		return "error"
	case "compressing":
		return "checkingUP"
	case "uploading":
		return "uploading"
	case "dead":
		return "error"
	default:
		return "unknown"
	}
}
