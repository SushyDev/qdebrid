package real_debrid

import (
	"fmt"
	"io"
	"net/http"
	"qdebrid/config"
	"strconv"
	"strings"
	"sync"
	"time"

	real_debrid "github.com/sushydev/real_debrid_go"
	real_debrid_api "github.com/sushydev/real_debrid_go/api"
)

type throttledTransport struct {
	Transport   http.RoundTripper
	mu          sync.Mutex
	lastRequest time.Time
	interval    time.Duration
}

var _ http.RoundTripper = (*throttledTransport)(nil)

func (t *throttledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()

	// Calculate how long to wait
	now := time.Now()
	elapsed := now.Sub(t.lastRequest)
	if elapsed < t.interval {
		// Wait the remaining time
		time.Sleep(t.interval - elapsed)
	}

	// Update the last request time
	t.lastRequest = time.Now()
	t.mu.Unlock()

	// Send the actual request using the underlying transport
	return t.Transport.RoundTrip(req)
}

func newThrottledTransport() *throttledTransport {
	return &throttledTransport{
		Transport: http.DefaultTransport,
		interval:  1 * time.Second,
	}
}

type Client struct {
	client *real_debrid.Client
}

var instance *Client

func GetClient() *Client {
	if instance == nil {
		instance = NewClient()
	}

	return instance
}

func NewClient() *Client {
	httpClient := &http.Client{
		Transport: newThrottledTransport(),
	}

	token := config.GetSettings().RealDebrid.Token

	realDebridClient := real_debrid.NewClient(token, httpClient)

	return &Client{
		client: realDebridClient,
	}
}

func (c *Client) AddTorrentByUrl(url string) (string, error) {
	if strings.HasPrefix(url, "magnet") {
		response, err := real_debrid_api.AddMagnet(c.client, url)
		if err != nil {
			return "", err
		}

		err = c.selectFiles(response.Id)
		if err != nil {
			return "", err
		}

		return response.Id, nil
	}

	if strings.HasPrefix(url, "http") {
		file, err := fetchTorrentFile(url)
		if err != nil {
			return "", err
		}

		response, err := real_debrid_api.AddTorrent(c.client, file)
		if err != nil {
			return "", err
		}

		file.Close()

		err = c.selectFiles(response.Id)
		if err != nil {
			return "", err
		}

		return response.Id, nil
	}

	return "", fmt.Errorf("unsupported URL: %s", url)
}

func (c *Client) AddTorrentByFile(file io.ReadCloser) (string, error) {
	response, err := real_debrid_api.AddTorrent(c.client, file)
	if err != nil {
		return "", err
	}

	file.Close()

	err = c.selectFiles(response.Id)
	if err != nil {
		return "", err
	}

	return response.Id, nil
}

func (c *Client) DeleteByHash(hash string) error {
	torrents, err := real_debrid_api.GetTorrents(c.client, 1000, 1)
	if err != nil {
		return err
	}

	id := getTorrentIdFromHash(torrents, hash)

	return real_debrid_api.Delete(c.client, id)
}

func (c *Client) Delete(id string) error {
	return real_debrid_api.Delete(c.client, id)
}

func (c *Client) GetFilesByHash(hash string) ([]real_debrid_api.TorrentFile, error) {
	torrents, err := real_debrid_api.GetTorrents(c.client, 1000, 1)
	if err != nil {
		return nil, err
	}

	torrentId := getTorrentIdFromHash(torrents, hash)
	if torrentId == "" {
		return nil, fmt.Errorf("torrent not found")
	}

	return c.GetFiles(torrentId)
}

func (c *Client) GetFiles(id string) ([]real_debrid_api.TorrentFile, error) {
	torrentInfo, err := real_debrid_api.GetTorrentInfo(c.client, id)
	if err != nil {
		return nil, err
	}

	return torrentInfo.Files, nil
}

func (c *Client) GetTorrentInfoByHash(hash string) (*real_debrid_api.TorrentInfo, error) {
	torrents, err := real_debrid_api.GetTorrents(c.client, 1000, 1)
	if err != nil {
		return nil, err
	}

	torrentId := getTorrentIdFromHash(torrents, hash)
	if torrentId == "" {
		return nil, fmt.Errorf("torrent not found")
	}

	return c.GetTorrentInfo(torrentId)
}

func (c *Client) GetTorrentInfo(id string) (*real_debrid_api.TorrentInfo, error) {
	return real_debrid_api.GetTorrentInfo(c.client, id)
}

func (c *Client) GetTorrents() (*real_debrid_api.Torrents, error) {
	return real_debrid_api.GetTorrents(c.client, 1000, 1)
}

func (c *Client) selectFiles(torrentId string) error {
	torrentInfo, err := real_debrid_api.GetTorrentInfo(c.client, torrentId)
	if err != nil {
		return err
	}

	allowedFileIds, err := getAllowedFileIds(torrentInfo)
	if err != nil {
		return err
	}

	fileIds := strings.Join(allowedFileIds, ",")

	return real_debrid_api.SelectFiles(c.client, torrentId, fileIds)
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

func getAllowedFileIds(torrentInfo *real_debrid_api.TorrentInfo) ([]string, error) {
	settings := config.GetSettings()

	if len(settings.QDebrid.AllowedFileTypes) == 0 {
		return []string{"all"}, nil
	}

	var ids []string
	for _, file := range torrentInfo.Files {
		if file.Bytes <= settings.QDebrid.MinFileSize {
			continue
		}

		for _, extension := range settings.QDebrid.AllowedFileTypes {
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

func getTorrentIdFromHash(torrents *real_debrid_api.Torrents, hash string) string {
	if torrents == nil {
		return ""
	}

	for _, torrent := range *torrents {
		if strings.EqualFold(torrent.Hash, hash) {
			return torrent.ID
		}
	}

	return ""
}

func MapRealDebridStatus(status string) string {
	switch status {
	case "magnet_error":
		return "error"
	case "magnet_conversion":
		return "checkingUP"
	case "waiting_files_selection":
		return "checkingUP"
	case "queued":
		return "checkingUP"
	case "downloading":
		return "downloading"
	case "downloaded":
		return "pausedUP"
	case "error":
		return "error"
	case "virus":
		return "error"
	case "compressing":
		return "checkingUP"
	case "uploading":
		return "uploading"
	case "dead":
		return "error"
	default:
		return "unknown"
	}
}
