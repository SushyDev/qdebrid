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

type Settings struct {
	CategoryName string     `yaml:"category_name"`
	SavePath     string     `yaml:"save_path"`
	RealDebrid   RealDebrid `yaml:"real_debrid"`
	Overseerr    Overseerr  `yaml:"overseerr"`
	Sonarr       Sonarr     `yaml:"sonarr"`
	Radarr       Radarr     `yaml:"radarr"`
}

type Config struct {
	Settings Settings `yaml:"settings"`
}
