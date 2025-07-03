#!/bin/bash
# Deployment script for SoulHound Discord Music Bot

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
IMAGE_NAME="soulhound"
TAG="latest"
CONTAINER_NAME="soulhound-bot"
CONTAINER_ENGINE="docker"
DEPLOYMENT_MODE="local"
ENV_FILE=".env"

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo "Options:"
    echo "  -i, --image     Image name (default: soulhound)"
    echo "  -t, --tag       Image tag (default: latest)"
    echo "  -n, --name      Container name (default: soulhound-bot)"
    echo "  -e, --env-file  Environment file (default: .env)"
    echo "  -m, --mode      Deployment mode: local|compose|swarm (default: local)"
    echo "  --docker        Use Docker (default)"
    echo "  --podman        Use Podman"
    echo "  --stop          Stop and remove existing container"
    echo "  --restart       Restart existing container"
    echo "  --logs          Show container logs"
    echo "  --status        Show container status"
    echo "  -h, --help      Display this help message"
}

# Parse command line arguments
ACTION="deploy"

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
        -n|--name)
            CONTAINER_NAME="$2"
            shift 2
            ;;
        -e|--env-file)
            ENV_FILE="$2"
            shift 2
            ;;
        -m|--mode)
            DEPLOYMENT_MODE="$2"
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
        --stop)
            ACTION="stop"
            shift
            ;;
        --restart)
            ACTION="restart"
            shift
            ;;
        --logs)
            ACTION="logs"
            shift
            ;;
        --status)
            ACTION="status"
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

# Function to check if container exists
container_exists() {
    $CONTAINER_ENGINE ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"
}

# Function to check if container is running
container_running() {
    $CONTAINER_ENGINE ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"
}

# Function to stop and remove container
stop_container() {
    echo -e "${YELLOW}Stopping and removing container: $CONTAINER_NAME${NC}"
    
    if container_exists; then
        if container_running; then
            $CONTAINER_ENGINE stop $CONTAINER_NAME
            echo -e "${GREEN}✅ Container stopped${NC}"
        fi
        $CONTAINER_ENGINE rm $CONTAINER_NAME
        echo -e "${GREEN}✅ Container removed${NC}"
    else
        echo -e "${YELLOW}⚠️  Container $CONTAINER_NAME does not exist${NC}"
    fi
}

# Function to deploy container
deploy_container() {
    echo -e "${YELLOW}Deploying SoulHound Discord Music Bot...${NC}"
    echo -e "${YELLOW}Container Engine: $CONTAINER_ENGINE${NC}"
    echo -e "${YELLOW}Image: $IMAGE_NAME:$TAG${NC}"
    echo -e "${YELLOW}Container Name: $CONTAINER_NAME${NC}"
    echo -e "${YELLOW}Environment File: $ENV_FILE${NC}"
    
    # Check if environment file exists
    if [ ! -f "$ENV_FILE" ]; then
        echo -e "${RED}Error: Environment file $ENV_FILE not found${NC}"
        echo -e "${YELLOW}Please create $ENV_FILE from .env.example${NC}"
        exit 1
    fi
    
    # Stop existing container if it exists
    if container_exists; then
        echo -e "${YELLOW}Existing container found. Stopping and removing...${NC}"
        stop_container
    fi
    
    # Deploy based on mode
    case $DEPLOYMENT_MODE in
        "local")
            deploy_local
            ;;
        "compose")
            deploy_compose
            ;;
        "swarm")
            deploy_swarm
            ;;
        *)
            echo -e "${RED}Unknown deployment mode: $DEPLOYMENT_MODE${NC}"
            exit 1
            ;;
    esac
}

# Function to deploy locally
deploy_local() {
    echo -e "${YELLOW}Deploying in local mode...${NC}"
    
    $CONTAINER_ENGINE run -d \
        --name "$CONTAINER_NAME" \
        --env-file "$ENV_FILE" \
        --restart unless-stopped \
        "$IMAGE_NAME:$TAG"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Container deployed successfully!${NC}"
        echo -e "${GREEN}Container Name: $CONTAINER_NAME${NC}"
        
        # Show container status
        sleep 2
        show_status
    else
        echo -e "${RED}❌ Deployment failed!${NC}"
        exit 1
    fi
}

# Function to deploy with docker-compose
deploy_compose() {
    echo -e "${YELLOW}Deploying with docker-compose...${NC}"
    
    if [ ! -f "docker-compose.yml" ]; then
        echo -e "${RED}Error: docker-compose.yml not found${NC}"
        exit 1
    fi
    
    $CONTAINER_ENGINE compose up -d
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Services deployed successfully!${NC}"
        $CONTAINER_ENGINE compose ps
    else
        echo -e "${RED}❌ Deployment failed!${NC}"
        exit 1
    fi
}

# Function to deploy with docker swarm
deploy_swarm() {
    echo -e "${YELLOW}Deploying with docker swarm...${NC}"
    echo -e "${RED}Docker Swarm deployment not implemented yet${NC}"
    exit 1
}

# Function to show container status
show_status() {
    echo -e "${BLUE}=== Container Status ===${NC}"
    
    if container_exists; then
        $CONTAINER_ENGINE ps -a --filter "name=$CONTAINER_NAME" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        
        if container_running; then
            echo -e "${GREEN}✅ Container is running${NC}"
        else
            echo -e "${RED}❌ Container is not running${NC}"
            echo -e "${YELLOW}Recent logs:${NC}"
            $CONTAINER_ENGINE logs --tail 10 $CONTAINER_NAME
        fi
    else
        echo -e "${RED}❌ Container $CONTAINER_NAME does not exist${NC}"
    fi
}

# Function to show logs
show_logs() {
    echo -e "${BLUE}=== Container Logs ===${NC}"
    
    if container_exists; then
        $CONTAINER_ENGINE logs -f $CONTAINER_NAME
    else
        echo -e "${RED}❌ Container $CONTAINER_NAME does not exist${NC}"
        exit 1
    fi
}

# Function to restart container
restart_container() {
    echo -e "${YELLOW}Restarting container: $CONTAINER_NAME${NC}"
    
    if container_exists; then
        $CONTAINER_ENGINE restart $CONTAINER_NAME
        echo -e "${GREEN}✅ Container restarted${NC}"
        sleep 2
        show_status
    else
        echo -e "${RED}❌ Container $CONTAINER_NAME does not exist${NC}"
        exit 1
    fi
}

# Main execution
case $ACTION in
    "deploy")
        deploy_container
        ;;
    "stop")
        stop_container
        ;;
    "restart")
        restart_container
        ;;
    "logs")
        show_logs
        ;;
    "status")
        show_status
        ;;
    *)
        echo -e "${RED}Unknown action: $ACTION${NC}"
        usage
        exit 1
        ;;
esac

echo -e "${GREEN}Operation completed! 🎉${NC}"