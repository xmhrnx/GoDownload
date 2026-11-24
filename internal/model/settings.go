package model

// Theme is the app's color mode.
type Theme string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
)

// PowerAction is what the computer does when downloads or a schedule finish.
type PowerAction string

const (
	PowerNone      PowerAction = ""
	PowerQuitApp   PowerAction = "quit"
	PowerSleep     PowerAction = "sleep"
	PowerHibernate PowerAction = "hibernate"
	PowerSignOut   PowerAction = "signout"
	PowerRestart   PowerAction = "restart"
	PowerShutdown  PowerAction = "shutdown"
)

// PowerActions lists every action in menu order.
var PowerActions = []PowerAction{
	PowerNone, PowerQuitApp, PowerSleep, PowerHibernate, PowerSignOut, PowerRestart, PowerShutdown,
}

// Title is the action's menu label.
func (a PowerAction) Title() string {
	switch a {
	case PowerQuitApp:
		return "Quit GoDownload"
	case PowerSleep:
		return "Sleep"
	case PowerHibernate:
		return "Hibernate"
	case PowerSignOut:
		return "Sign out"
	case PowerRestart:
		return "Restart"
	case PowerShutdown:
		return "Shut down"
	default:
		return "Do nothing"
	}
}

// Settings are the user's preferences.
type Settings struct {
	Theme          Theme `json:"theme"`
	CloseToTray    bool  `json:"closeToTray"`
	StartMinimized bool  `json:"startMinimized"`
	Notifications  bool  `json:"notifications"`
	WatchClipboard bool  `json:"watchClipboard"`

	// BaseDir is the root download folder; each category gets a subfolder.
	BaseDir string `json:"baseDir"`
	// CategoryDirs overrides a category's folder.
	CategoryDirs map[Category]string `json:"categoryDirs,omitempty"`
	// AskWhereToSave shows the add dialog for every new download, including
	// ones the browser sends.
	AskWhereToSave bool `json:"askWhereToSave"`

	MaxActive     int    `json:"maxActive"`
	Connections   int    `json:"connections"`
	DownloadLimit int64  `json:"downloadLimit"` // bytes per second, all downloads together
	UploadLimit   int64  `json:"uploadLimit"`   // bytes per second, all torrents together
	Retries       int    `json:"retries"`
	Proxy         string `json:"proxy,omitempty"`

	TorrentPort   int     `json:"torrentPort"`
	SeedByDefault bool    `json:"seedByDefault"`
	SeedRatio     float64 `json:"seedRatio"`
	DHT           bool    `json:"dht"`
	PortForward   bool    `json:"portForward"`
	ExtraTrackers bool    `json:"extraTrackers"`
	Encryption    bool    `json:"encryption"`

	// Player is the id of the player "Open now" starts (see package player).
	Player        string `json:"player"`
	PlayerCommand string `json:"playerCommand,omitempty"`
	// StreamBufferMB is how much is fetched ahead of the play position.
	StreamBufferMB int `json:"streamBufferMB"`

	APIPort  int    `json:"apiPort"`
	APIToken string `json:"apiToken,omitempty"`
	// CaptureBrowser lets the extension hand browser downloads to GoDownload.
	CaptureBrowser bool `json:"captureBrowser"`

	// WhenDone runs once every download has finished.
	WhenDone PowerAction `json:"whenDone,omitempty"`

	// DetailsHeight is the details panel's height in dp, as last dragged.
	DetailsHeight int `json:"detailsHeight,omitempty"`

	Schedules []Schedule `json:"schedules,omitempty"`
}

// DefaultSettings returns the settings a fresh install starts with.
func DefaultSettings(downloadsDir string) Settings {
	return Settings{
		Theme:          ThemeSystem,
		CloseToTray:    true,
		Notifications:  true,
		BaseDir:        downloadsDir,
		MaxActive:      4,
		Connections:    8,
		Retries:        10,
		TorrentPort:    42069,
		SeedByDefault:  true,
		SeedRatio:      2,
		DHT:            true,
		PortForward:    true,
		ExtraTrackers:  true,
		Encryption:     true,
		Player:         "auto",
		StreamBufferMB: 16,
		APIPort:        47821,
		CaptureBrowser: true,
	}
}
