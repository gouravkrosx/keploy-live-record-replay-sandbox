#!/bin/bash

# ═══════════════════════════════════════════════════════════════════════════════
# E-Commerce Marketplace API - Complete Cleanup Script
# This script removes ALL Docker resources related to this project
# ═══════════════════════════════════════════════════════════════════════════════

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo "  E-Commerce Marketplace API - Complete Cleanup"
echo "═══════════════════════════════════════════════════════════════════════════════"
echo -e "${NC}"

# Confirmation prompt
echo -e "${YELLOW}⚠️  WARNING: This will remove ALL project-related Docker resources:${NC}"
echo "   • Stop and remove all containers (marketplace-api, marketplace-mysql, marketplace-seeder)"
echo "   • Remove Docker volumes (mysql_data)"
echo "   • Remove Docker networks (marketplace-network)"
echo "   • Remove Docker images (app, seeder)"
echo "   • Remove any dangling images/volumes"
echo ""

read -p "Are you sure you want to continue? (y/N): " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Cleanup cancelled.${NC}"
    exit 0
fi

echo ""
echo -e "${BLUE}🧹 Starting cleanup...${NC}"
echo ""

# Step 1: Stop and remove containers using docker compose
echo -e "${YELLOW}[1/7] Stopping and removing Docker Compose services...${NC}"
if docker compose ps -q 2>/dev/null | grep -q .; then
    docker compose --profile seed down -v --remove-orphans 2>/dev/null || true
    echo -e "${GREEN}   ✓ Docker Compose services removed${NC}"
else
    docker compose down -v --remove-orphans 2>/dev/null || true
    echo -e "${GREEN}   ✓ No running services found${NC}"
fi

# Step 2: Stop and remove any lingering project containers
echo -e "${YELLOW}[2/7] Removing any remaining project containers...${NC}"
for container in marketplace-api marketplace-mysql marketplace-seeder; do
    if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
        docker stop "$container" 2>/dev/null || true
        docker rm -f "$container" 2>/dev/null || true
        echo -e "${GREEN}   ✓ Removed container: ${container}${NC}"
    fi
done
echo -e "${GREEN}   ✓ Container cleanup complete${NC}"

# Step 3: Remove project volumes
echo -e "${YELLOW}[3/7] Removing project volumes...${NC}"
for volume in keploy-live-record-replay-sandbox_mysql_data mysql_data; do
    if docker volume ls -q | grep -q "^${volume}$"; then
        docker volume rm "$volume" 2>/dev/null || true
        echo -e "${GREEN}   ✓ Removed volume: ${volume}${NC}"
    fi
done
echo -e "${GREEN}   ✓ Volume cleanup complete${NC}"

# Step 4: Remove project networks
echo -e "${YELLOW}[4/7] Removing project networks...${NC}"
for network in keploy-live-record-replay-sandbox_marketplace-network marketplace-network; do
    if docker network ls -q --filter name="$network" | grep -q .; then
        docker network rm "$network" 2>/dev/null || true
        echo -e "${GREEN}   ✓ Removed network: ${network}${NC}"
    fi
done
echo -e "${GREEN}   ✓ Network cleanup complete${NC}"

# Step 5: Remove project images
echo -e "${YELLOW}[5/7] Removing project Docker images...${NC}"
for image in keploy-live-record-replay-sandbox-app keploy-live-record-replay-sandbox-seeder; do
    if docker images -q "$image" 2>/dev/null | grep -q .; then
        docker rmi -f "$image" 2>/dev/null || true
        echo -e "${GREEN}   ✓ Removed image: ${image}${NC}"
    fi
done
echo -e "${GREEN}   ✓ Image cleanup complete${NC}"

# Step 6: Remove dangling images and volumes
echo -e "${YELLOW}[6/7] Removing dangling Docker resources...${NC}"
docker image prune -f 2>/dev/null || true
docker volume prune -f 2>/dev/null || true
echo -e "${GREEN}   ✓ Dangling resources cleaned${NC}"

# Step 7: Remove local build artifacts
echo -e "${YELLOW}[7/7] Removing local build artifacts...${NC}"
if [ -f "./main" ]; then
    rm -f ./main
    echo -e "${GREEN}   ✓ Removed ./main binary${NC}"
fi
if [ -f "./.env" ]; then
    rm -f ./.env
    echo -e "${GREEN}   ✓ Removed ./.env file${NC}"
fi
echo -e "${GREEN}   ✓ Build artifacts cleaned${NC}"

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  ✅ Cleanup completed successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}To start fresh, run:${NC}"
echo ""
echo "  # Docker setup:"
echo "  docker compose up -d"
echo "  docker compose --profile seed run --rm seeder"
echo ""
echo "  # Native setup:"
echo "  cp .env.example .env"
echo "  go run cmd/seeder/main.go"
echo "  go run cmd/server/main.go"
echo ""
