package config

type RealDebrid struct {
	Token   string `yaml:"token"`
	Timeout int    `yaml:"timeout"`
}

type Overseerr struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token"`
}

type Sonarr struct {
	Token string `yaml:"token"`
}

type Radarr struct {
	Token string `yaml:"token"`
}

type Settings struct {
	RealDebrid RealDebrid `yaml:"real_debrid"`
	Overseerr  Overseerr  `yaml:"overseerr"`
	Sonarr     Sonarr     `yaml:"sonarr"`
	Radarr     Radarr     `yaml:"radarr"`
}
type Config struct {
	Settings Settings `yaml:"settings"`
}
