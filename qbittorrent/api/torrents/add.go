package torrents

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

func GetEntries(r *http.Request) (entries, error) {
	entries := entries{}

	validateAndParseForm(r)

	contentType := parseContentType(r)

	urls := r.FormValue("urls")
	if urls != "" {
		lines := splitString(urls)
		for _, url := range lines {
			entries.urls = append(entries.urls, url)
		}
	}

	if contentType == "multipart/form-data" {
		files := r.MultipartForm.File["torrents"]
		for _, fileHeader := range files {
			file, err := processFile(fileHeader)
			if err != nil {
				return entries, fmt.Errorf("failed to process file: %s, error: %v", fileHeader.Filename, err)
			}

			entries.files = append(entries.files, file)
		}
	}

	return entries, nil
}

func (module *Module) Add(entries entries) ([]byte, error) {
	logger := module.GetLogger()

	logger.Info("Received request to add torrent(s)")

	addedUrlTorrentIds, err := module.addFromUrls(entries.urls)
	if err != nil {
		return nil, fmt.Errorf("Failed to add torrents from urls: %v", err)
	}

	addedFileIds, err := module.addFromFiles(entries.files)
	if err != nil {
		return nil, fmt.Errorf("Failed to add torrents from files: %v", err)
	}

	for _, torrentId := range addedUrlTorrentIds {
		err = module.selectFiles(torrentId)
		if err != nil {
			return nil, fmt.Errorf("Failed to select files: %v", err)
		}
	}

	for _, torrentId := range addedFileIds {
		err = module.selectFiles(torrentId)
		if err != nil {
			return nil, fmt.Errorf("Failed to select files: %v", err)
		}
	}

	added := len(addedUrlTorrentIds) + len(addedFileIds)

	logger.Info(fmt.Sprintf("Added %d torrents", added))

	return []byte(fmt.Sprintf("Successfully added %d torrent(s)", added)), nil
}

// --- Helpers

func (module *Module) addFromUrls(urls []string) ([]string, error) {
	torrentIds := make([]string, 0)

	for _, url := range urls {
		if strings.HasPrefix(url, "magnet") {
			response, err := real_debrid_api.AddMagnet(module.RealDebridClient, url)
			if err != nil {
				return torrentIds, err
			}

			torrentIds = append(torrentIds, response.Id)

			continue
		}

		if strings.HasPrefix(url, "http") {
			file, err := fetchTorrentFile(url)
			if err != nil {
				return torrentIds, err
			}

			response, err := real_debrid_api.AddTorrent(module.RealDebridClient, file)
			if err != nil {
				return torrentIds, err
			}

			file.Close()

			torrentIds = append(torrentIds, response.Id)

			continue
		}

		return torrentIds, fmt.Errorf("unsupported URL format: %s", url)
	}

	return torrentIds, nil
}

func (module *Module) addFromFiles(files []io.ReadCloser) ([]string, error) {
	torrentIds := make([]string, 0)

	for _, file := range files {
		response, err := real_debrid_api.AddTorrent(module.RealDebridClient, file)
		if err != nil {
			return torrentIds, err
		}

		torrentIds = append(torrentIds, response.Id)
	}

	return torrentIds, nil
}

func (module *Module) selectFiles(torrentId string) error {
	torrentInfo, err := real_debrid_api.GetTorrentInfo(module.RealDebridClient, torrentId)
	if err != nil {
		return err
	}

	allowedFileIds, err := module.getAllowedFileIds(torrentInfo)
	if err != nil {
		return err
	}

	fileIds := strings.Join(allowedFileIds, ",")

	real_debrid_api.SelectFiles(module.RealDebridClient, torrentId, fileIds)

	return nil
}

func (module *Module) getAllowedFileIds(torrentInfo *real_debrid_api.TorrentInfo) ([]string, error) {
	if len(module.Settings.QDebrid.AllowedFileTypes) == 0 {
		return []string{"all"}, nil
	}

	var ids []string
	for _, file := range torrentInfo.Files {
		if file.Bytes <= module.Settings.QDebrid.MinFileSize {
			continue
		}

		for _, extension := range module.Settings.QDebrid.AllowedFileTypes {
			if !strings.HasSuffix(file.Path, extension) {
				continue
			}

			ids = append(ids, strconv.Itoa(file.ID))
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("No accepted files found")
	}

	return ids, nil
}

func fetchTorrentFile(url string) (io.ReadCloser, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch torrent: status code %d", response.StatusCode)
	}

	return response.Body, nil
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

type entries struct {
	urls  []string
	files []io.ReadCloser
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
