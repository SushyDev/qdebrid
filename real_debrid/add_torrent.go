package real_debrid

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func AddTorrent(torrent io.Reader, files string) error {
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

	switch response.StatusCode {
	case 201:
		var data AddMagnetResponse
		err = json.NewDecoder(response.Body).Decode(&data)
		if err != nil {
			return err
		}

		return selectFiles(data.Id)
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
		_, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("[%v] Unknown error", response.StatusCode)
	}
}
