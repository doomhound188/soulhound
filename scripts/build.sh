#!/bin/bash
# Build script for SoulHound Discord Music Bot

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
IMAGE_NAME="soulhound"
TAG="latest"
DOCKERFILE="Dockerfile"
BUILD_CONTEXT="."

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -i, --image     Image name (default: soulhound)"
    echo "  -t, --tag       Image tag (default: latest)"
    echo "  -f, --file      Dockerfile path (default: Dockerfile)"
    echo "  -c, --context   Build context (default: .)"
    echo "  --docker        Use Docker (default)"
    echo "  --podman        Use Podman"
    echo "  -h, --help      Display this help message"
}

# Parse command line arguments
CONTAINER_ENGINE="docker"

while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--image)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -f|--file)
            DOCKERFILE="$2"
            shift 2
            ;;
        -c|--context)
            BUILD_CONTEXT="$2"
            shift 2
            ;;
        --docker)
            CONTAINER_ENGINE="docker"
            shift
            ;;
        --podman)
            CONTAINER_ENGINE="podman"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

# Check if container engine is available
if ! command -v $CONTAINER_ENGINE &> /dev/null; then
    echo -e "${RED}Error: $CONTAINER_ENGINE is not installed or not in PATH${NC}"
    exit 1
fi

echo -e "${YELLOW}Building SoulHound Discord Music Bot...${NC}"
echo -e "${YELLOW}Container Engine: $CONTAINER_ENGINE${NC}"
echo -e "${YELLOW}Image: $IMAGE_NAME:$TAG${NC}"
echo -e "${YELLOW}Dockerfile: $DOCKERFILE${NC}"
echo -e "${YELLOW}Build Context: $BUILD_CONTEXT${NC}"

# Build the image
echo -e "${YELLOW}Starting build...${NC}"
if $CONTAINER_ENGINE build -t "$IMAGE_NAME:$TAG" -f "$DOCKERFILE" "$BUILD_CONTEXT"; then
    echo -e "${GREEN}✅ Build completed successfully!${NC}"
    echo -e "${GREEN}Image: $IMAGE_NAME:$TAG${NC}"
    
    # Display image info
    echo -e "${YELLOW}Image information:${NC}"
    $CONTAINER_ENGINE images "$IMAGE_NAME:$TAG"
else
    echo -e "${RED}❌ Build failed!${NC}"
    exit 1
fi

# Optional: Run basic smoke test
echo -e "${YELLOW}Running basic smoke test...${NC}"
if $CONTAINER_ENGINE run --rm "$IMAGE_NAME:$TAG" --help 2>/dev/null; then
    echo -e "${GREEN}✅ Smoke test passed!${NC}"
else
    echo -e "${YELLOW}⚠️  Smoke test skipped (help command not available)${NC}"
fi

echo -e "${GREEN}Build completed successfully! 🎉${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Set up your environment variables in .env file"
echo -e "  2. Run: $CONTAINER_ENGINE run --env-file .env $IMAGE_NAME:$TAG"
echo -e "  3. Or use: $CONTAINER_ENGINE compose up -d"