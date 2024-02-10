package qbittorrent

import (
	"fmt"
	"net/http"
	"qdebrid/config"
	"qdebrid/qbittorrent/app"
	"qdebrid/qbittorrent/auth"
	"qdebrid/qbittorrent/torrents"
)

var apiPath = "/api/v2"

var settings = config.GetSettings()

func Listen() {
	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc(apiPath+"/auth/login", auth.Login)

	// App
	mux.HandleFunc(apiPath+"/app/webapiVersion", app.Version)
	mux.HandleFunc(apiPath+"/app/preferences", app.Preferences)

	// Torrents
	mux.HandleFunc(apiPath+"/torrents/categories", torrents.Categories)
	mux.HandleFunc(apiPath+"/torrents/add", torrents.Add)
	mux.HandleFunc(apiPath+"/torrents/delete", torrents.Delete)
	mux.HandleFunc(apiPath+"/torrents/info", torrents.Info)
	mux.HandleFunc(apiPath+"/torrents/files", torrents.Files)
	mux.HandleFunc(apiPath+"/torrents/properties", torrents.Properties)

	host := ""
	port := "8080"

	if settings.QDebrid.Host != "" {
		host = settings.QDebrid.Host
	}

	if settings.QDebrid.Port != 0 {
		port = fmt.Sprintf("%d", settings.QDebrid.Port)
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	http.ListenAndServe(addr, mux)
}
