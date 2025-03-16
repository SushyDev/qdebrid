package torrents

import (
	"qdebrid/config"
)

type Module struct {
}

func New() *Module {
	return &Module{}
}

var settings = config.GetSettings()
