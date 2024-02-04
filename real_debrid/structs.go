package real_debrid

type InstantAvailabilityFile struct {
	FileName string `json:"filename"`
	FileSize int    `json:"filesize"`
}

// TODO: Idk if this is quite correct yet
type InstantAvailabilityResponse map[string]map[string][]map[string]InstantAvailabilityFile

type Torrent struct {
	ID       string   `json:"id"`
	Filename string   `json:"filename"`
	Hash     string   `json:"hash"`
	Bytes    int      `json:"bytes"`
	Host     string   `json:"host"`
	Split    int      `json:"split"`
	Progress float64  `json:"progress"`
	Status   string   `json:"status"`
	Added    string   `json:"added"`
	Links    []string `json:"links"`
	Ended    string   `json:"ended"`
}

type TorrentsResponse []Torrent

type File struct {
	ID       int    `json:"id"`
	Path     string `json:"path"`
	Bytes    int    `json:"bytes"`
	Selected int    `json:"selected"`
}

type TorrentInfoResponse struct {
	ID               string    `json:"id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	Hash             string    `json:"hash"`
	Bytes            int       `json:"bytes"`
	OriginalBytes    int       `json:"original_bytes"`
	Host             string    `json:"host"`
	Split            int       `json:"split"`
	Progress         int       `json:"progress"`
	Status           string    `json:"status"`
	Added            string    `json:"added"`
	Files            []File    `json:"files"`
	Links            []string  `json:"links"`
	Ended            string    `json:"ended,omitempty"`
	Speed            int       `json:"speed,omitempty"`
	Seeders          int       `json:"seeders,omitempty"`
}
