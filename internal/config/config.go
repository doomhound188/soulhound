package config

type Config struct {
	DiscordToken  string
	YouTubeToken  string
	SpotifyToken  string
	DefaultPlayer string // "yt" or "sp"
}

type PlayerSettings struct {
	SmartPlayEnabled bool
	Platform         string
	Volume           int
}

var (
	AppConfig    Config
	PlayerConfig PlayerSettings
)

func Init(discordToken, youtubeToken, spotifyToken string) {
	AppConfig = Config{
		DiscordToken:  discordToken,
		YouTubeToken:  youtubeToken,
		SpotifyToken:  spotifyToken,
		DefaultPlayer: "yt",
	}

	PlayerConfig = PlayerSettings{
		SmartPlayEnabled: false,
		Platform:         AppConfig.DefaultPlayer,
		Volume:           100,
	}
}

func SetDefaultPlayer(platform string) {
	if platform == "yt" || platform == "sp" {
		AppConfig.DefaultPlayer = platform
		PlayerConfig.Platform = platform
	}
}

func ToggleSmartPlay(enabled bool) {
	PlayerConfig.SmartPlayEnabled = enabled
}
