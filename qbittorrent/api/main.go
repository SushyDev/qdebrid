package api

import (
	"qdebrid/config"
	"qdebrid/logger"

	real_debrid "github.com/sushydev/real_debrid_go"

	"go.uber.org/zap"
)

type Api struct {
	Settings         config.Settings
	RealDebridClient *real_debrid.Client
}

func New() *Api {
	settings := config.GetSettings()

	client := real_debrid.NewClient(settings.RealDebrid.Token)

	return &Api{
		Settings:         settings,
		RealDebridClient: client,
	}
}

func (module *Api) GetLogger() *zap.SugaredLogger {
	return logger.Sugar()
}
