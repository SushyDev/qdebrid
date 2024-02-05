package config

type RealDebrid struct {
	Token string `yaml:"token"`
}

type Overseerr struct {
	Host  string `yaml:"host"`
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
	Port          int        `yaml:"port"`
	CategoryName  string     `yaml:"category_name"`
	SavePath      string     `yaml:"save_path"`
	ValidatePaths bool       `yaml:"validate_paths"`
	RealDebrid    RealDebrid `yaml:"real_debrid"`
	Overseerr     Overseerr  `yaml:"overseerr"`
	Sonarr        Sonarr     `yaml:"sonarr"`
	Radarr        Radarr     `yaml:"radarr"`
	Zurg          Zurg       `yaml:"zurg"`
}

type Config struct {
	Settings Settings `yaml:"settings"`
}
