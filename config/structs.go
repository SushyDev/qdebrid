package config

type Settings struct {
	QDebrid    QDebrid    `yaml:"qdebrid"`
	RealDebrid RealDebrid `yaml:"real_debrid"`
}

type Config struct {
	Settings Settings `yaml:"settings"`
}

type QDebrid struct {
	Host string `yaml:"host" default:""`
	Port int    `yaml:"port" default:"8080"`

	ResponseCacheTTL string `yaml:"response_cache_ttl" default:"5m"`

	CategoryName string `yaml:"category_name" default:"qdebrid"`
	SavePath     string `yaml:"save_path" default:"/mnt/debrid_drive/new_torrents"`

	ValidatePaths bool `yaml:"validate_paths" default:"true"`

	AllowUncached    bool     `yaml:"allow_uncached" default:"false"`
	AllowedFileTypes []string `yaml:"allowed_file_types" default:"[mkv, mp4]"`

	MinFileSize int `yaml:"min_file_size" default:"500000000"` // In Bytes (500MB)

	LogLevel string `yaml:"log_level" default:"info"`
}

type RealDebrid struct {
	Token string `yaml:"token"`
}
