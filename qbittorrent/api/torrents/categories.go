package torrents

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type category struct {
	Name     string `json:"name"`
	SavePath string `json:"savePath"`
}

type categories map[string]category

func list() categories {
	return map[string]category{
		"main": {
			Name:     "main",
			SavePath: "",
		},
	}
}

func (module Module) Categories(w http.ResponseWriter, r *http.Request) {
	logger := module.GetLogger()

	logger.Info("Received request for torrent categories")

	categories := list()

	jsonData, err := json.Marshal(categories)
	if err != nil {
		logger.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Debug(fmt.Sprintf("Categories: %s", jsonData))

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}
