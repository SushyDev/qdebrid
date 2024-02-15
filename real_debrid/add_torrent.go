package real_debrid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"qdebrid/logger"
)

func AddTorrent(torrent io.Reader) error {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/addTorrent"

	req, err := http.NewRequest("PUT", url.String(), torrent)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	err = nil
	switch response.StatusCode {
	case 201:
		var data AddMagnetResponse
		if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
			return err
		}

		sugar.Debug(logger.EndpointMessage("real_debrid", "AddTorrent", fmt.Sprintf("Torrent ID: %v", data.Id)))

		err = selectFiles(data.Id)
	case 400:
		err = fmt.Errorf("Bad Request (see error message)")
	case 401:
		err = fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		err = fmt.Errorf("Permission denied (account locked, not premium) or Infringing torrent")
	case 503:
		err = fmt.Errorf("Service unavailable (see error message)")
	case 504:
		err = fmt.Errorf("Service timeout (see error message)")
	default:
		err = fmt.Errorf("[%v] Unknown error", response.StatusCode)
	}

	return err
}
