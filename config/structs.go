package config

type RealDebrid struct {
	Token string `yaml:"token"`
}

type Sonarr struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

type Radarr struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

type Zurg struct {
	Host string `yaml:"host"`
}

type Settings struct {
	Host          string     `yaml:"host"`
	Port          int        `yaml:"port"` // 8080 by default
	CategoryName  string     `yaml:"category_name"` // Download Client category name
	SavePath      string     `yaml:"save_path"` // Debrid mount path (/mnt/zurg/__all__)
	ValidatePaths bool       `yaml:"validate_paths"` // Ping zurg to check if path exists
	RealDebrid    RealDebrid `yaml:"real_debrid"`
	Sonarr        Sonarr     `yaml:"sonarr"`
	Radarr        Radarr     `yaml:"radarr"`
	Zurg          Zurg       `yaml:"zurg"`
}

type Config struct {
	Settings Settings `yaml:"settings"`
}
