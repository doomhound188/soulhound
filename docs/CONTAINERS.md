# SoulHound Container Documentation

This document provides detailed information about the containerization setup for SoulHound Discord Music Bot.

## Overview

SoulHound now supports containerization with both Docker and Podman, providing easy deployment options for users who prefer containerized applications.

## Container Features

- **Multi-stage builds** for optimized production images
- **Non-root user execution** for security
- **Health checks** for monitoring
- **Multi-architecture support** (amd64, arm64)
- **Automated CI/CD** with GitHub Actions
- **Development and production** configurations

## Quick Start

### Prerequisites

- Docker or Podman installed
- Discord Bot Token (minimum requirement)
- YouTube/Spotify tokens (optional)

### Option 1: Using Pre-built Images

```bash
# Pull the latest image
docker pull ghcr.io/doomhound188/soulhound:latest

# Create environment file
cat > .env << EOF
DISCORD_TOKEN=your_discord_token_here
YOUTUBE_TOKEN=your_youtube_token_here
SPOTIFY_TOKEN=your_spotify_token_here
EOF

# Run the container
docker run -d --name soulhound-bot \
  --env-file .env \
  --restart unless-stopped \
  ghcr.io/doomhound188/soulhound:latest
```

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/doomhound188/soulhound.git
cd soulhound

# Setup environment
make setup
# Edit .env file with your tokens

# Build and deploy
make docker-build
make docker-deploy
```

### Option 3: Docker Compose

```bash
# Clone repository
git clone https://github.com/doomhound188/soulhound.git
cd soulhound

# Setup environment
cp .env.example .env
# Edit .env file with your tokens

# Deploy
docker compose up -d
```

## Scripts Documentation

### build.sh

Builds container images with various options:

```bash
./scripts/build.sh [OPTIONS]
```

**Options:**
- `-i, --image`: Image name (default: soulhound)
- `-t, --tag`: Image tag (default: latest)
- `-f, --file`: Dockerfile path (default: Dockerfile)
- `-c, --context`: Build context (default: .)
- `--docker`: Use Docker (default)
- `--podman`: Use Podman
- `-h, --help`: Display help

**Examples:**
```bash
# Build with Docker
./scripts/build.sh

# Build with Podman
./scripts/build.sh --podman

# Build with custom tag
./scripts/build.sh -t my-soulhound:v1.0

# Build with custom Dockerfile
./scripts/build.sh -f Dockerfile.custom
```

### test.sh

Tests container images and application:

```bash
./scripts/test.sh [OPTIONS]
```

**Tests performed:**
1. Image existence check
2. Container creation and startup
3. Go unit tests
4. Security checks (non-root user)
5. Image size reporting
6. Health check validation

### deploy.sh

Deploys and manages containers:

```bash
./scripts/deploy.sh [OPTIONS]
```

**Deployment modes:**
- `local`: Single container deployment
- `compose`: Docker Compose deployment
- `swarm`: Docker Swarm deployment (planned)

**Management commands:**
- `--stop`: Stop and remove container
- `--restart`: Restart container
- `--logs`: Show container logs
- `--status`: Show container status

## Docker vs Podman

Both container engines are fully supported:

| Feature | Docker | Podman |
|---------|--------|--------|
| Basic functionality | ✅ | ✅ |
| Rootless mode | ⚠️ | ✅ |
| Compose support | ✅ | ✅ |
| Swarm mode | ✅ | ❌ |
| Build caching | ✅ | ✅ |

### Podman-specific Notes

Podman runs containers rootlessly by default, which provides better security:

```bash
# All scripts support Podman with --podman flag
./scripts/build.sh --podman
./scripts/test.sh --podman
./scripts/deploy.sh --podman

# Podman compose
podman compose up -d
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DISCORD_TOKEN` | Yes | Discord Bot Token |
| `YOUTUBE_TOKEN` | No | YouTube Data API Token |
| `SPOTIFY_TOKEN` | No | Spotify API Token |

## Troubleshooting

### Common Issues

1. **Container exits immediately**
   - Check that DISCORD_TOKEN is set correctly
   - Verify token has proper bot permissions

2. **Permission denied errors**
   - Ensure container runs as non-root user
   - Check file permissions if using volumes

3. **Build failures**
   - Verify network connectivity
   - Check Docker/Podman installation

4. **Audio issues**
   - Audio streaming requires additional setup
   - Consider using external audio processing

### Debugging

```bash
# Check container logs
./scripts/deploy.sh --logs

# Run container interactively
docker run -it --rm soulhound:latest sh

# Check container status
./scripts/deploy.sh --status

# Run tests
./scripts/test.sh
```

For more detailed information, see the main README.md file.