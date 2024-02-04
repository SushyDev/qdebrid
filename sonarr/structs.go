package sonarr

import "time"

type HistoryResponse struct {
	Page          int      `json:"page"`
	PageSize      int      `json:"pageSize"`
	SortKey       string   `json:"sortKey"`
	SortDirection string   `json:"sortDirection"`
	TotalRecords  int      `json:"totalRecords"`
	Records       []Record `json:"records"`
}

type Language struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Quality struct {
	Quality struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Source     string `json:"source"`
		Resolution int    `json:"resolution"`
	} `json:"quality"`
	Revision struct {
		Version  int  `json:"version"`
		Real     int  `json:"real"`
		IsRepack bool `json:"isRepack"`
	} `json:"revision"`
}

type Specification struct {
	ID                 int      `json:"id"`
	Name               string   `json:"name"`
	Implementation     string   `json:"implementation"`
	ImplementationName string   `json:"implementationName"`
	InfoLink           string   `json:"infoLink"`
	Negate             bool     `json:"negate"`
	Required           bool     `json:"required"`
	Fields             []Field  `json:"fields"`
	Presets            []string `json:"presets"`
}

type CustomFormat struct {
	ID                              int             `json:"id"`
	Name                            string          `json:"name"`
	IncludeCustomFormatWhenRenaming bool            `json:"includeCustomFormatWhenRenaming"`
	Specifications                  []Specification `json:"specifications"`
}

type Field struct {
	Order                       int            `json:"order"`
	Name                        string         `json:"name"`
	Label                       string         `json:"label"`
	Unit                        string         `json:"unit"`
	HelpText                    string         `json:"helpText"`
	HelpTextWarning             string         `json:"helpTextWarning"`
	HelpLink                    string         `json:"helpLink"`
	Value                       string         `json:"value"`
	Type                        string         `json:"type"`
	Advanced                    bool           `json:"advanced"`
	SelectOptions               []SelectOption `json:"selectOptions"`
	SelectOptionsProviderAction string         `json:"selectOptionsProviderAction"`
	Section                     string         `json:"section"`
	Hidden                      string         `json:"hidden"`
	Privacy                     string         `json:"privacy"`
	Placeholder                 string         `json:"placeholder"`
	IsFloat                     bool           `json:"isFloat"`
}

type SelectOption struct {
	Value int    `json:"value"`
	Name  string `json:"name"`
	Order int    `json:"order"`
	Hint  string `json:"hint"`
}

type Data struct {
	Indexer            string    `json:"indexer"`
	NzbInfoUrl         string    `json:"nzbInfoUrl"`
	ReleaseGroup       string    `json:"releaseGroup"`
	Age                string    `json:"age"`
	AgeHours           string    `json:"ageHours"`
	AgeMinutes         string    `json:"ageMinutes"`
	PublishedDate      time.Time `json:"publishedDate"`
	DownloadClient     string    `json:"downloadClient"`
	DownloadClientName string    `json:"downloadClientName"`
	Size               string    `json:"size"`
	DownloadUrl        string    `json:"downloadUrl"`
	GUID               string    `json:"guid"`
	TvdbId             string    `json:"tvdbId"`
	TvRageId           string    `json:"tvRageId"`
	Protocol           string    `json:"protocol"`
	CustomFormatScore  string    `json:"customFormatScore"`
	SeriesMatchType    string    `json:"seriesMatchType"`
	ReleaseSource      string    `json:"releaseSource"`
	TorrentInfoHash    string    `json:"torrentInfoHash"`
}

type Record struct {
	ID                  int            `json:"id"`
	EpisodeID           int            `json:"episodeId"`
	SeriesID            int            `json:"seriesId"`
	SourceTitle         string         `json:"sourceTitle"`
	Languages           []Language     `json:"languages"`
	Quality             Quality        `json:"quality"`
	CustomFormats       []CustomFormat `json:"customFormats"`
	CustomFormatScore   int            `json:"customFormatScore"`
	QualityCutoffNotMet bool           `json:"qualityCutoffNotMet"`
	Date                time.Time      `json:"date"`
	DownloadID          string         `json:"downloadId"`
	EventType           string         `json:"eventType"`
	Data                Data           `json:"data"`
	Episode             Episode        `json:"episode"`
	Series              Series         `json:"series"`
}

type EpisodeFile struct {
	ID                  int            `json:"id"`
	SeriesID            int            `json:"seriesId"`
	SeasonNumber        int            `json:"seasonNumber"`
	RelativePath        string         `json:"relativePath"`
	Path                string         `json:"path"`
	Size                int            `json:"size"`
	DateAdded           time.Time      `json:"dateAdded"`
	SceneName           string         `json:"sceneName"`
	ReleaseGroup        string         `json:"releaseGroup"`
	Languages           []Language     `json:"languages"`
	Quality             Quality        `json:"quality"`
	CustomFormats       []CustomFormat `json:"customFormats"`
	CustomFormatScore   int            `json:"customFormatScore"`
	MediaInfo           MediaInfo      `json:"mediaInfo"`
	QualityCutoffNotMet bool           `json:"qualityCutoffNotMet"`
}

type Episode struct {
	ID                         int         `json:"id"`
	SeriesID                   int         `json:"seriesId"`
	TVDBID                     int         `json:"tvdbId"`
	EpisodeFileID              int         `json:"episodeFileId"`
	SeasonNumber               int         `json:"seasonNumber"`
	EpisodeNumber              int         `json:"episodeNumber"`
	Title                      string      `json:"title"`
	AirDate                    string      `json:"airDate"`
	AirDateUTC                 time.Time   `json:"airDateUtc"`
	Runtime                    int         `json:"runtime"`
	FinaleType                 string      `json:"finaleType"`
	Overview                   string      `json:"overview"`
	EpisodeFile                EpisodeFile `json:"episodeFile"`
	HasFile                    bool        `json:"hasFile"`
	Monitored                  bool        `json:"monitored"`
	AbsoluteEpisodeNumber      int         `json:"absoluteEpisodeNumber"`
	SceneAbsoluteEpisodeNumber int         `json:"sceneAbsoluteEpisodeNumber"`
	SceneEpisodeNumber         int         `json:"sceneEpisodeNumber"`
	SceneSeasonNumber          int         `json:"sceneSeasonNumber"`
	UnverifiedSceneNumbering   bool        `json:"unverifiedSceneNumbering"`
	EndTime                    time.Time   `json:"endTime"`
	GrabDate                   time.Time   `json:"grabDate"`
	SeriesTitle                string      `json:"seriesTitle"`
	Images                     []struct {
		CoverType string `json:"coverType"`
		URL       string `json:"url"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
	Grabbed bool `json:"grabbed"`
}

type MediaInfo struct {
	ID                    int    `json:"id"`
	AudioBitrate          int    `json:"audioBitrate"`
	AudioChannels         int    `json:"audioChannels"`
	AudioCodec            string `json:"audioCodec"`
	AudioLanguages        string `json:"audioLanguages"`
	AudioStreamCount      int    `json:"audioStreamCount"`
	VideoBitDepth         int    `json:"videoBitDepth"`
	VideoBitrate          int    `json:"videoBitrate"`
	VideoCodec            string `json:"videoCodec"`
	VideoFps              int    `json:"videoFps"`
	VideoDynamicRange     string `json:"videoDynamicRange"`
	VideoDynamicRangeType string `json:"videoDynamicRangeType"`
	Resolution            string `json:"resolution"`
	RunTime               string `json:"runTime"`
	ScanType              string `json:"scanType"`
	Subtitles             string `json:"subtitles"`
}

type Series struct {
	ID              int              `json:"id"`
	Title           string           `json:"title"`
	AlternateTitles []AlternateTitle `json:"alternateTitles"`
	SortTitle       string           `json:"sortTitle"`
	Status          string           `json:"status"`
	Ended           bool             `json:"ended"`
	ProfileName     string           `json:"profileName"`
	Overview        string           `json:"overview"`
	NextAiring      time.Time        `json:"nextAiring"`
	PreviousAiring  time.Time        `json:"previousAiring"`
	Network         string           `json:"network"`
	AirTime         string           `json:"airTime"`
	Images          []struct {
		CoverType string `json:"coverType"`
		URL       string `json:"url"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
	OriginalLanguage  Language  `json:"originalLanguage"`
	RemotePoster      string    `json:"remotePoster"`
	Seasons           []Season  `json:"seasons"`
	Year              int       `json:"year"`
	Path              string    `json:"path"`
	QualityProfileID  int       `json:"qualityProfileId"`
	SeasonFolder      bool      `json:"seasonFolder"`
	Monitored         bool      `json:"monitored"`
	MonitorNewItems   string    `json:"monitorNewItems"`
	UseSceneNumbering bool      `json:"useSceneNumbering"`
	Runtime           int       `json:"runtime"`
	TVDBID            int       `json:"tvdbId"`
	TvRageID          int       `json:"tvRageId"`
	TvMazeID          int       `json:"tvMazeId"`
	FirstAired        time.Time `json:"firstAired"`
	LastAired         time.Time `json:"lastAired"`
	SeriesType        string    `json:"seriesType"`
	CleanTitle        string    `json:"cleanTitle"`
	ImdbID            string    `json:"imdbId"`
	TitleSlug         string    `json:"titleSlug"`
	RootFolderPath    string    `json:"rootFolderPath"`
	Folder            string    `json:"folder"`
	Certification     string    `json:"certification"`
	Genres            []string  `json:"genres"`
	Tags              []int     `json:"tags"`
	Added             time.Time `json:"added"`
	AddOptions        struct {
		IgnoreEpisodesWithFiles      bool   `json:"ignoreEpisodesWithFiles"`
		IgnoreEpisodesWithoutFiles   bool   `json:"ignoreEpisodesWithoutFiles"`
		Monitor                      string `json:"monitor"`
		SearchForMissingEpisodes     bool   `json:"searchForMissingEpisodes"`
		SearchForCutoffUnmetEpisodes bool   `json:"searchForCutoffUnmetEpisodes"`
	} `json:"addOptions"`
	Ratings struct {
		Votes int `json:"votes"`
		Value int `json:"value"`
	} `json:"ratings"`
	Statistics struct {
		SeasonCount       int      `json:"seasonCount"`
		EpisodeFileCount  int      `json:"episodeFileCount"`
		EpisodeCount      int      `json:"episodeCount"`
		TotalEpisodeCount int      `json:"totalEpisodeCount"`
		SizeOnDisk        int      `json:"sizeOnDisk"`
		ReleaseGroups     []string `json:"releaseGroups"`
		PercentOfEpisodes int      `json:"percentOfEpisodes"`
	} `json:"statistics"`
	EpisodesChanged bool `json:"episodesChanged"`
}

type AlternateTitle struct {
	Title             string `json:"title"`
	SeasonNumber      int    `json:"seasonNumber"`
	SceneSeasonNumber int    `json:"sceneSeasonNumber"`
	SceneOrigin       string `json:"sceneOrigin"`
	Comment           string `json:"comment"`
}

type Season struct {
	SeasonNumber int              `json:"seasonNumber"`
	Monitored    bool             `json:"monitored"`
	Statistics   SeasonStatistics `json:"statistics"`
	Images       []struct {
		CoverType string `json:"coverType"`
		URL       string `json:"url"`
		RemoteURL string `json:"remoteUrl"`
	} `json:"images"`
}

type SeasonStatistics struct {
	NextAiring        time.Time `json:"nextAiring"`
	PreviousAiring    time.Time `json:"previousAiring"`
	EpisodeFileCount  int       `json:"episodeFileCount"`
	EpisodeCount      int       `json:"episodeCount"`
	TotalEpisodeCount int       `json:"totalEpisodeCount"`
	SizeOnDisk        int       `json:"sizeOnDisk"`
	ReleaseGroups     []string  `json:"releaseGroups"`
	PercentOfEpisodes int       `json:"percentOfEpisodes"`
}
