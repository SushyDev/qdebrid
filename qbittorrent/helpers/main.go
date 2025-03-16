package helpers

import (
	"os"
	"path/filepath"
	"qdebrid/config"
)

var settings = config.GetSettings()

func pathExists(path string) (bool, error) {
	directory := filepath.Join(settings.QDebrid.SavePath, path)

	_, err := os.Stat(directory)
	if err != nil {
		return false, nil
	}

	return true, nil
}
