package torrents

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"qdebrid/cache"
	"qdebrid/debrid/client/real_debrid"
	"strings"
)

func Add(w http.ResponseWriter, r *http.Request, c *cache.Cache) {
	urls, files, err := getEntries(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var ids []string

	client := real_debrid.GetClient()

	for _, url := range urls {
		id, err := client.AddTorrentByUrl(url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ids = append(ids, id)
	}

	for _, file := range files {
		id, err := client.AddTorrentByFile(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ids = append(ids, id)
	}

	c.Clear()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok."))
}

func getEntries(r *http.Request) ([]string, []io.ReadCloser, error) {
	validateAndParseForm(r)

	var returnUrls []string
	var returnFiles []io.ReadCloser

	contentType := parseContentType(r)

	urls := r.FormValue("urls")
	if urls != "" {
		lines := splitString(urls)
		for _, url := range lines {
			returnUrls = append(returnUrls, url)
		}
	}

	if contentType == "multipart/form-data" {
		files := r.MultipartForm.File["torrents"]
		for _, fileHeader := range files {
			file, err := processFile(fileHeader)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to process file: %s, error: %v", fileHeader.Filename, err)
			}

			returnFiles = append(returnFiles, file)
		}
	}

	return returnUrls, returnFiles, nil
}

func processFile(fileHeader *multipart.FileHeader) (io.ReadCloser, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %v", err)
	}

	return file, nil
}

func splitString(input string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result
}

func validateAndParseForm(r *http.Request) error {
	contentType := parseContentType(r)

	switch contentType {
	case "multipart/form-data":
		return r.ParseMultipartForm(0)
	case "application/x-www-form-urlencoded":
		return r.ParseForm()
	default:
		return fmt.Errorf("unsupported Content-Type: %s", contentType)
	}
}

func parseContentType(r *http.Request) string {
	contentHeader := r.Header.Get("Content-Type")
	parts := strings.Split(contentHeader, ";")

	return parts[0]
}
