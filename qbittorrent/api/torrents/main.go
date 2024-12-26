package torrents

import (
	"qdebrid/config"
	"qdebrid/qbittorrent/api"
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
