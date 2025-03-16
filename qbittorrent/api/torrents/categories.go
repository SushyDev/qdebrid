package torrents

import (
	"encoding/json"
	"net/http"
)

type category struct {
	Name     string `json:"name"`
	SavePath string `json:"savePath"`
}

type categories map[string]category

func Categories(w http.ResponseWriter, r *http.Request) {
	categories := list()

	jsonData, err := json.Marshal(categories)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func list() categories {
	categoryName := settings.QDebrid.CategoryName

	return map[string]category{
		categoryName: {
			Name:     categoryName,
			SavePath: settings.QDebrid.SavePath,
		},
	}
}
