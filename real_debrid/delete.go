package real_debrid

import (
	"fmt"
	"net/http"
	"net/url"
)

func Delete(id string) error {
	url, _ := url.Parse(apiHost)
	url.Path += apiPath + "/torrents/delete/" + id

	req, err := http.NewRequest("DELETE", url.String(), nil)
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
	case 204:
		return nil
	case 401:
		return fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		return fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		return fmt.Errorf("Unknown resource (invalid id)")
	default:
		return fmt.Errorf("Unknown error")
	}
}
