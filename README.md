# SoulHound Discord Music Bot

A powerful Discord music bot that can play music from both YouTube and Spotify, written in Go.

## Features

- Multi-platform support (YouTube and Spotify)
- Smart playlist recommendations based on genre
- Queue management system
- Platform-specific commands with prefix support (yt: or sp:)
- Default platform preferences
- Voice channel management

## Prerequisites

- Go 1.24 or higher
- Discord Bot Token
- YouTube Data API Token (optional)
- Spotify API Token (optional)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/doomhound/soulhound.git
cd soulhound
```

2. Build the project:
```bash
go build -o soulhound cmd/main.go
```

## Configuration

Set up your API tokens either through environment variables or command-line flags:

Environment variables:
```bash
export DISCORD_TOKEN='your_discord_token'
export YOUTUBE_TOKEN='your_youtube_token'  # Optional
export SPOTIFY_TOKEN='your_spotify_token'  # Optional
```

Or use command-line flags when running the bot:
```bash
./soulhound -discord=your_discord_token -youtube=your_youtube_token -spotify=your_spotify_token
```

## Commands

- `!play <query>` - Play a song (prefix with yt: or sp: to specify platform)
- `!pause` - Pause current playback
- `!resume` - Resume paused playback
- `!stop` - Stop playback and clear queue
- `!queue` - Show current queue
- `!skip` - Skip to next track
- `!remove <number>` - Remove track from queue
- `!search <query>` - Search without adding to queue
- `!setdefault <yt/sp>` - Set default platform
- `!smartplay <on/off>` - Toggle smart recommendations

Examples:
```bash
!play yt:never gonna give you up
!play sp:shape of you
!setdefault yt
!smartplay on
```

## Development

The project structure follows standard Go project layout:

```
.
├── cmd/
│   └── main.go
├── internal/
│   ├── audio/
│   ├── bot/
│   ├── config/
│   └── queue/
└── go.mod
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request