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

	// Recommended to leave default. Category name for the *Arrs
	CategoryName string `yaml:"category_name" default:"qdebrid"`

	// Parent directory where all your media gets added
	SavePath string `yaml:"save_path" default:"/mnt/fvs/debrid_drive/media_manager"`

	// Recommended to leave on true unless unavoidable. Checks if the files are available on disk and updates the status accordingly for the *Arrs
	ValidatePaths bool `yaml:"validate_paths" default:"true"`

	// Monentarily deprecated: Checks the instantAvailability api, reject if not available
	AllowUncached bool `yaml:"allow_uncached" default:"false"`

	// Allowed file types to be considered to select when adding in RD
	AllowedFileTypes []string `yaml:"allowed_file_types" default:"[mkv, mp4]"`

	// Minimum file size to be considered to select when adding in RD
	MinFileSize int `yaml:"min_file_size" default:"500000000"` // In Bytes (500MB)

	LogLevel string `yaml:"log_level" default:"info"`
}

type RealDebrid struct {
	Token string `yaml:"token"`
}
