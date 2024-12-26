package torrents

import (
	"encoding/json"
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
			SavePath: settings.QDebrid.SavePath,
		},
	}
}

func Categories() ([]byte, error) {
	categories := list()

	jsonData, err := json.Marshal(categories)
	if err != nil {
		return nil, err
	}

	return jsonData, err
}
