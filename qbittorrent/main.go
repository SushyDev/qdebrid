package qbittorrent

import (
	"fmt"
	"net/http"
	"qdebrid/qbittorrent/app"
	"qdebrid/qbittorrent/auth"
	"qdebrid/qbittorrent/torrents"
)

func Listen() {
	mux := http.NewServeMux()

	// Routes

	// Auth
	mux.HandleFunc("/api/v2/auth/login", auth.Login)

	// App
	mux.HandleFunc("/api/v2/app/webapiVersion", app.Version)
	mux.HandleFunc("/api/v2/app/preferences", app.Preferences)

	// Torrents
	mux.HandleFunc("/api/v2/torrents/categories", torrents.Categories)
	mux.HandleFunc("/api/v2/torrents/add", torrents.Add)
	mux.HandleFunc("/api/v2/torrents/delete", torrents.Delete)
	mux.HandleFunc("/api/v2/torrents/info", torrents.Info)
	mux.HandleFunc("/api/v2/torrents/files", torrents.Files)
	mux.HandleFunc("/api/v2/torrents/properties", torrents.Properties)

	// TODO Make configurable
	fmt.Println("Server listening on :8080")
	http.ListenAndServe("localhost:8080", mux)
}
