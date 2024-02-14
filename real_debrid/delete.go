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

	err = nil
	switch response.StatusCode {
	case 204:
		err = nil
	case 401:
		err = fmt.Errorf("Bad token (expired, invalid)")
	case 403:
		err = fmt.Errorf("Permission denied (account locked, not premium)")
	case 404:
		err = fmt.Errorf("Unknown resource (invalid id)")
	default:
		err = fmt.Errorf("Unknown error")
	}

	return err
}

func DeleteBecauseError(id string, err error) error {
	if err := Delete(id); err != nil {
		return err
	}

	return err
}
