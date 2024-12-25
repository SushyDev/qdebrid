package qbittorrent

import (
	"fmt"
	"net/http"
	"qdebrid/qbittorrent/api"
	"qdebrid/qbittorrent/api/app"
	"qdebrid/qbittorrent/api/auth"
	"qdebrid/qbittorrent/api/torrents"
)

var apiPath = "/api/v2"

// todo move to /api dir ??
func Listen() {
	mux := http.NewServeMux()

	apiInstance := api.New()

	authModule := auth.New(apiInstance)
	appModule := app.New(apiInstance)
	torrentsModule := torrents.New(apiInstance)

	// Auth
	mux.HandleFunc(apiPath+"/auth/login", authModule.Login)

	// App
	mux.HandleFunc(apiPath+"/app/webapiVersion", appModule.Version)
	mux.HandleFunc(apiPath+"/app/preferences", appModule.Preferences)

	// Torrents
	mux.HandleFunc(apiPath+"/torrents/add", torrentsModule.Add)
	mux.HandleFunc(apiPath+"/torrents/categories", torrentsModule.Categories)
	mux.HandleFunc(apiPath+"/torrents/delete", torrentsModule.Delete)
	mux.HandleFunc(apiPath+"/torrents/files", torrentsModule.Files)
	mux.HandleFunc(apiPath+"/torrents/info", torrentsModule.Info)
	mux.HandleFunc(apiPath+"/torrents/properties", torrentsModule.Properties)

	host := ""
	port := "8080"

	if apiInstance.Settings.QDebrid.Host != "" {
		host = apiInstance.Settings.QDebrid.Host
	}

	if apiInstance.Settings.QDebrid.Port != 0 {
		port = fmt.Sprintf("%d", apiInstance.Settings.QDebrid.Port)
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	apiInstance.GetLogger().Info("Listening on ", addr)
	http.ListenAndServe(addr, mux)
}
