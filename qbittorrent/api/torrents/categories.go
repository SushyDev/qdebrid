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
	categoryName := settings.QDebrid.CategoryName

	return map[string]category{
		categoryName: {
			Name:     categoryName,
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
