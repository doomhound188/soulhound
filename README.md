# SoulHound Discord Music Bot

A powerful Discord music bot that can play music from both YouTube and Spotify, written in Go.

## Features

- Multi-platform support (YouTube and Spotify)
- Smart playlist recommendations based on genre
- Queue management system
- Platform-specific commands with prefix support (yt: or sp:)
- Default platform preferences
- Voice channel management
- 🐳 **Docker & Podman support with easy deployment**

## Prerequisites

- Go 1.21 or higher (for development)
- Discord Bot Token
- YouTube Data API Token (optional, falls back to mock data)
- Spotify API Token (optional, falls back to mock data)
- **Docker or Podman (for containerized deployment)**

## Recent Updates

- ✅ Fixed module import path issues
- ✅ Implemented working audio provider search functionality
- ✅ Added graceful error handling and fallbacks
- ✅ Voice channel requirement now only applies to music commands
- ✅ Added comprehensive help system
- ✅ Improved concurrent access safety
- ✅ Added mock data fallbacks for testing without API keys
- 🆕 **Added Docker/Podman containerization support**
- 🆕 **Added automated build, test, and deployment scripts**
- 🆕 **Added GitHub Actions for CI/CD**

## Quick Start with Docker/Podman

### Option 1: Using Pre-built Scripts (Recommended)

1. **Clone the repository:**
```bash
git clone https://github.com/doomhound188/soulhound.git
cd soulhound
```

2. **Set up environment variables:**
```bash
cp .env.example .env
# Edit .env file with your tokens
```

3. **Build the container:**
```bash
# Using Docker
./scripts/build.sh

# Using Podman
./scripts/build.sh --podman
```

4. **Run tests:**
```bash
# Using Docker
./scripts/test.sh

# Using Podman
./scripts/test.sh --podman
```

5. **Deploy the bot:**
```bash
# Using Docker
./scripts/deploy.sh

# Using Podman
./scripts/deploy.sh --podman
```

### Option 2: Using Docker Compose

1. **Clone and setup:**
```bash
git clone https://github.com/doomhound188/soulhound.git
cd soulhound
cp .env.example .env
# Edit .env with your tokens
```

2. **Deploy with Docker Compose:**
```bash
docker compose up -d
```

3. **View logs:**
```bash
docker compose logs -f
```

4. **Stop the bot:**
```bash
docker compose down
```

### Option 3: Using Container Registries

Pull and run the latest image from GitHub Container Registry:

```bash
# Using Docker
docker run -d --name soulhound-bot \
  --env-file .env \
  --restart unless-stopped \
  ghcr.io/doomhound188/soulhound:latest

# Using Podman
podman run -d --name soulhound-bot \
  --env-file .env \
  --restart unless-stopped \
  ghcr.io/doomhound188/soulhound:latest
```

## Traditional Installation (Development)

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

### Environment File (.env) - Recommended for Containers
```bash
# Copy the example file
cp .env.example .env

# Edit the .env file with your actual tokens:
DISCORD_TOKEN=your_discord_token_here
YOUTUBE_TOKEN=your_youtube_token_here  # Optional
SPOTIFY_TOKEN=your_spotify_token_here  # Optional
```

### Environment Variables (Traditional)
```bash
export DISCORD_TOKEN='your_discord_token'
export YOUTUBE_TOKEN='your_youtube_token'  # Optional
export SPOTIFY_TOKEN='your_spotify_token'  # Optional
```

### Command-line Flags
```bash
./soulhound -discord=your_discord_token -youtube=your_youtube_token -spotify=your_spotify_token
```

## Container Management

### Build Scripts

The repository includes helpful scripts for container management:

- **`./scripts/build.sh`** - Build container images
- **`./scripts/test.sh`** - Test container images and application
- **`./scripts/deploy.sh`** - Deploy and manage containers

### Build Script Options
```bash
./scripts/build.sh [OPTIONS]
  -i, --image     Image name (default: soulhound)
  -t, --tag       Image tag (default: latest)
  --docker        Use Docker (default)
  --podman        Use Podman
```

### Deploy Script Options
```bash
./scripts/deploy.sh [OPTIONS]
  -m, --mode      Deployment mode: local|compose|swarm
  --stop          Stop and remove existing container
  --restart       Restart existing container
  --logs          Show container logs
  --status        Show container status
```

### Docker vs Podman

Both Docker and Podman are supported. Simply add `--podman` to any script to use Podman instead of Docker:

```bash
# Docker (default)
./scripts/build.sh
./scripts/deploy.sh

# Podman
./scripts/build.sh --podman
./scripts/deploy.sh --podman
```

## CI/CD and Automated Builds

The repository includes GitHub Actions that automatically:

- ✅ Run tests on every pull request
- ✅ Build container images on every push to main
- ✅ Push images to GitHub Container Registry
- ✅ Support multi-architecture builds (amd64, arm64)
- ✅ Run security scans with Trivy

### Available Container Images

Images are automatically built and published to GitHub Container Registry:

- `ghcr.io/doomhound188/soulhound:latest` - Latest from main branch
- `ghcr.io/doomhound188/soulhound:main` - Main branch
- `ghcr.io/doomhound188/soulhound:v1.0.0` - Version tags (when released)

## Commands

- `!help` - Show all available commands and usage examples
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
!help
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
│   └── main.go                 # Application entry point
├── internal/
│   ├── audio/                  # Audio provider implementations
│   ├── bot/                    # Discord bot logic
│   ├── config/                 # Configuration management
│   └── queue/                  # Music queue management
├── scripts/                    # Build, test, and deployment scripts
│   ├── build.sh               # Container build script
│   ├── test.sh                # Testing script
│   └── deploy.sh              # Deployment script
├── .github/workflows/          # GitHub Actions CI/CD
├── Dockerfile                  # Container image definition
├── docker-compose.yml          # Docker Compose configuration
├── .env.example               # Environment variables template
└── go.mod
```

### Development with Containers

For development, you can use the container environment:

```bash
# Build development image
./scripts/build.sh -t soulhound:dev

# Run development container with volume mount
docker run -it --rm \
  -v $(pwd):/app \
  -w /app \
  golang:1.21-alpine \
  sh

# Or use the development compose file
docker compose -f docker-compose.yml -f docker-compose.dev.yml up
```

### Using the Makefile

A convenient Makefile is provided for common tasks:

```bash
# See all available targets
make help

# Traditional development
make build          # Build Go binary
make test           # Run Go tests
make clean          # Clean build artifacts

# Docker workflow
make docker-build   # Build Docker image
make docker-test    # Test Docker image
make docker-deploy  # Deploy with Docker
make docker-logs    # Show container logs
make docker-stop    # Stop container

# Podman workflow
make podman-build   # Build Podman image
make podman-test    # Test Podman image
make podman-deploy  # Deploy with Podman

# Docker Compose
make compose-up     # Start with docker-compose
make compose-down   # Stop docker-compose
make compose-logs   # Show logs

# Quick setup
make setup          # Copy .env.example to .env
```

### Contributing to Container Setup

When contributing container-related changes:

1. Test with both Docker and Podman
2. Ensure scripts work on different platforms
3. Update documentation for any new features
4. Test the GitHub Actions workflow

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request