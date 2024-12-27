package torrents

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"qdebrid/config"
	"qdebrid/qbittorrent/api"
	"strings"
)

type Module struct {
	*api.Api
}

func New(api *api.Api) *Module {
	return &Module{
		Api: api,
	}
}

var settings = config.GetSettings()

func GetHash(request *http.Request) (string, error) {
	err := request.ParseForm()
	if err != nil {
		return "", err
	}

	return request.FormValue("hash"), nil
}

func GetHashes(request *http.Request) ([]string, error) {
	err := request.ParseForm()
	if err != nil {
		return nil, err
	}

	hashes := request.FormValue("hashes")
	return strings.Split(hashes, "|"), nil
}

func DecodeAuthHeader(request *http.Request) (string, string, error) {
	header := request.Header.Get("Authorization")
	if header == "" {
		return "", "", fmt.Errorf("Authorization header is missing")
	}

	encodedToken := strings.Split(header, " ")[1]

	bytes, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return "", "", err
	}

	bearer := string(bytes)

	colonIndex := strings.LastIndex(bearer, ":")
	host := bearer[:colonIndex]
	token := bearer[colonIndex+1:]

	return host, token, nil
}
