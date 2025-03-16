package qbittorrent

import (
	"fmt"
	"net/http"
	"qdebrid/cache"
	"qdebrid/config"
	"qdebrid/logger"
	"qdebrid/qbittorrent/api/app"
	"qdebrid/qbittorrent/api/auth"
	"qdebrid/qbittorrent/api/torrents"
)

var apiPath = "/api/v2"

type HandlerFunc func() []byte

var settings = config.GetSettings()

func registerHandler(mux *http.ServeMux, path string, handler HandlerFunc) {
	fmt.Println("Registering handler for ", path)

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		response := handler()

		w.WriteHeader(http.StatusOK)
		w.Write(response)
	})
}

func Listen() {
	mux := http.NewServeMux()

	cacheStore := cache.NewCache()

	logger := logger.Sugar()

	// Auth
	registerHandler(mux, fmt.Sprintf("%s%s", apiPath, "/auth/login"), auth.Login)

	// App
	registerHandler(mux, fmt.Sprintf("%s%s", apiPath, "/app/webapiVersion"), app.Version)
	registerHandler(mux, fmt.Sprintf("%s%s", apiPath, "/app/preferences"), app.Preferences)

	mux.HandleFunc(apiPath+"/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		torrents.Add(w, r, cacheStore)
	})

	mux.HandleFunc(apiPath+"/torrents/categories", func(w http.ResponseWriter, r *http.Request) {
		torrents.Categories(w, r)
	})

	mux.HandleFunc(apiPath+"/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		torrents.Delete(w, r, cacheStore)
	})

	mux.HandleFunc(apiPath+"/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		torrents.Files(w, r, cacheStore)
	})

	mux.HandleFunc(apiPath+"/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		torrents.Info(w, r, cacheStore)
	})

	mux.HandleFunc(apiPath+"/torrents/properties", func(w http.ResponseWriter, r *http.Request) {
		torrents.Properties(w, r, cacheStore)
	})

	host := ""
	port := "8080"

	if settings.QDebrid.Host != "" {
		host = settings.QDebrid.Host
	}

	if settings.QDebrid.Port != 0 {
		port = fmt.Sprintf("%d", settings.QDebrid.Port)
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	logger.Info("Listening on ", addr)
	http.ListenAndServe(addr, mux)
}
