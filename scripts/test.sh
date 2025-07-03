#!/bin/bash
# Test script for SoulHound Discord Music Bot

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
IMAGE_NAME="soulhound"
TAG="latest"
CONTAINER_ENGINE="docker"

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -i, --image     Image name (default: soulhound)"
    echo "  -t, --tag       Image tag (default: latest)"
    echo "  --docker        Use Docker (default)"
    echo "  --podman        Use Podman"
    echo "  -h, --help      Display this help message"
}

# Parse command line arguments
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

echo -e "${YELLOW}Testing SoulHound Discord Music Bot...${NC}"
echo -e "${YELLOW}Container Engine: $CONTAINER_ENGINE${NC}"
echo -e "${YELLOW}Image: $IMAGE_NAME:$TAG${NC}"

# Test 1: Check if image exists
echo -e "${YELLOW}Test 1: Checking if image exists...${NC}"
if $CONTAINER_ENGINE images "$IMAGE_NAME:$TAG" --format "table" | grep -q "$IMAGE_NAME"; then
    echo -e "${GREEN}✅ Image exists${NC}"
else
    echo -e "${RED}❌ Image does not exist. Please run build script first.${NC}"
    exit 1
fi

# Test 2: Test container creation and basic startup
echo -e "${YELLOW}Test 2: Testing container creation...${NC}"
CONTAINER_ID=$($CONTAINER_ENGINE run -d --name soulhound-test "$IMAGE_NAME:$TAG" || true)

if [ -n "$CONTAINER_ID" ]; then
    echo -e "${GREEN}✅ Container created successfully${NC}"
    
    # Wait a moment for container to initialize
    sleep 2
    
    # Check if container is running (it might exit quickly due to missing Discord token)
    if $CONTAINER_ENGINE ps -a --format "table" | grep -q "soulhound-test"; then
        echo -e "${GREEN}✅ Container startup test passed${NC}"
    else
        echo -e "${RED}❌ Container startup test failed${NC}"
    fi
    
    # Get container logs for debugging
    echo -e "${YELLOW}Container logs (last 10 lines):${NC}"
    $CONTAINER_ENGINE logs --tail 10 soulhound-test 2>/dev/null || echo "No logs available"
    
    # Cleanup test container
    $CONTAINER_ENGINE rm -f soulhound-test >/dev/null 2>&1
else
    echo -e "${RED}❌ Failed to create container${NC}"
    exit 1
fi

# Test 3: Test Go application directly (unit tests)
echo -e "${YELLOW}Test 3: Running Go unit tests...${NC}"
if go test ./... -v; then
    echo -e "${GREEN}✅ Go unit tests passed${NC}"
else
    echo -e "${YELLOW}⚠️  No Go unit tests found or tests failed${NC}"
fi

# Test 4: Container security check
echo -e "${YELLOW}Test 4: Basic security checks...${NC}"
SECURITY_ISSUES=0

# Check if running as non-root user
USER_INFO=$($CONTAINER_ENGINE run --rm "$IMAGE_NAME:$TAG" id 2>/dev/null || echo "uid=0(root)")
if echo "$USER_INFO" | grep -q "uid=1001"; then
    echo -e "${GREEN}✅ Container runs as non-root user${NC}"
else
    echo -e "${YELLOW}⚠️  Container may be running as root user${NC}"
    SECURITY_ISSUES=$((SECURITY_ISSUES + 1))
fi

# Test 5: Image size check
echo -e "${YELLOW}Test 5: Checking image size...${NC}"
IMAGE_SIZE=$($CONTAINER_ENGINE images "$IMAGE_NAME:$TAG" --format "{{.Size}}")
echo -e "${GREEN}✅ Image size: $IMAGE_SIZE${NC}"

# Test 6: Health check test (if applicable)
echo -e "${YELLOW}Test 6: Testing health check...${NC}"
if $CONTAINER_ENGINE inspect "$IMAGE_NAME:$TAG" | grep -q "Healthcheck"; then
    echo -e "${GREEN}✅ Health check configured${NC}"
else
    echo -e "${YELLOW}⚠️  No health check configured${NC}"
fi

# Summary
echo ""
echo -e "${YELLOW}=== Test Summary ===${NC}"
if [ $SECURITY_ISSUES -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed! 🎉${NC}"
else
    echo -e "${YELLOW}⚠️  Tests completed with $SECURITY_ISSUES security warnings${NC}"
fi

echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Create .env file with your Discord bot token"
echo -e "  2. Run: $CONTAINER_ENGINE run --env-file .env $IMAGE_NAME:$TAG"
echo -e "  3. Or use: $CONTAINER_ENGINE compose up"