package real_debrid

import (
	"fmt"
	"net/http"
	"qdebrid/config"
	"qdebrid/logger"
)

type RealDebridClient struct {
	http.Client
	Token string
}

func NewRealDebridClient(token string) *RealDebridClient {
	return &RealDebridClient{
		Client: http.Client{},
		Token:  token,
	}
}

func (c *RealDebridClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	return c.Client.Do(req)
}

type AddMagnetResponse struct {
	Id  string `json:"id"`
	Uri string `json:"uri"`
}

var apiHost = "https://api.real-debrid.com"
var apiPath = "/rest/1.0"

var settings = config.GetSettings()

var sugar = logger.Sugar()

var client = NewRealDebridClient(settings.RealDebrid.Token)
