package qbittorrent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"qdebrid/cache"
	"qdebrid/logger"
	"qdebrid/config"
	"qdebrid/qbittorrent/api/app"
	"qdebrid/qbittorrent/api/auth"
	"qdebrid/qbittorrent/api/torrents"
	"sort"
	"strings"
	"time"

	real_debrid "github.com/sushydev/real_debrid_go"
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

	client := real_debrid.NewClient(settings.RealDebrid.Token)

	logger := logger.Sugar()

	// Auth
	registerHandler(mux, fmt.Sprintf("%s%s", apiPath, "/auth/login"), auth.Login)

	// App
	registerHandler(mux, fmt.Sprintf("%s%s", apiPath, "/app/webapiVersion"), app.Version)
	registerHandler(mux, fmt.Sprintf("%s%s", apiPath, "/app/preferences"), app.Preferences)

	// Torrents --- TODO CACHE CERTAIN ENDPOINTS - CREATE CACHEKEY BASED ON ENDPOINT URL AND FORMDATA
	mux.HandleFunc(apiPath+"/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		entries, err := torrents.GetEntries(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response, err := torrents.Add(entries)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cacheStore.Clear()

		w.WriteHeader(http.StatusOK)
		w.Write(response)
	})

	mux.HandleFunc(apiPath+"/torrents/categories", func(w http.ResponseWriter, r *http.Request) {
		categories, err := torrents.Categories()
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(categories)
	})

	mux.HandleFunc(apiPath+"/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		hashes, err := torrents.GetHashes(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, hash := range hashes {
			err = torrents.Delete(client, hash)
			if err != nil {
				logger.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		cacheStore.Clear()

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc(apiPath+"/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		cacheKey, err := getCacheKey(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cachedFiles := cacheStore.Get(cacheKey)
		if cachedFiles != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(cachedFiles)
			return
		}

		hash, err := torrents.GetHash(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		files, err := torrents.Files(client, hash)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cacheStore.Store(cacheKey, cache.Entry{
			Value:      files,
			Expiration: time.Now().Add(15 * time.Minute),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(files)
	})

	mux.HandleFunc(apiPath+"/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		host, token, err := torrents.DecodeAuthHeader(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		info, err := torrents.Info(client, cacheStore, host, token)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(info)
	})

	mux.HandleFunc(apiPath+"/torrents/properties", func(w http.ResponseWriter, r *http.Request) {
		cacheKey, err := getCacheKey(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cachedProperties := cacheStore.Get(cacheKey)
		if cachedProperties != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(cachedProperties)
			return
		}

		hash, err := torrents.GetHash(r)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		properties, err := torrents.Properties(client, hash)
		if err != nil {
			logger.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cacheStore.Store(cacheKey, cache.Entry{
			Value:      properties,
			Expiration: time.Now().Add(15 * time.Minute),
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(properties)
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

func getCacheKey(r *http.Request) (string, error) {
	parsedUrl, err := url.Parse(r.URL.String())
	if err != nil {
		return "", err
	}

	err = r.ParseForm()
	if err != nil {
		return "", err
	}

	var params []string
	for key, values := range r.Form {
		for _, value := range values {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
	}

	sort.Strings(params)

	key := fmt.Sprintf("%s?%s", parsedUrl.Path, strings.Join(params, "&"))

	hash := sha256.New()
	hash.Write([]byte(key))
	hashSum := hash.Sum(nil)

	cacheKey := hex.EncodeToString(hashSum)

	return cacheKey, nil
}
