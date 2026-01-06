#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

NAMESPACE="live"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Default: build fresh images
USE_CACHE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --use-cache)
            USE_CACHE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --use-cache    Skip docker compose build, use existing cached images"
            echo "  -h, --help     Show this help message"
            echo ""
            echo "By default, images are rebuilt fresh using 'docker compose build'."
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information."
            exit 1
            ;;
    esac
done

# Timer function
start_timer() {
    TIMER_START=$(date +%s)
}

show_elapsed() {
    local end=$(date +%s)
    local elapsed=$((end - TIMER_START))
    echo -e "${CYAN}(${elapsed}s elapsed)${NC}"
}

# Wait with spinner and timer
wait_with_spinner() {
    local pid=$1
    local message=$2
    local spinner=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
    local i=0
    local start=$(date +%s)
    
    while kill -0 $pid 2>/dev/null; do
        local elapsed=$(($(date +%s) - start))
        printf "\r${YELLOW}${spinner[$i]} ${message} (${elapsed}s)${NC}  "
        i=$(( (i+1) % ${#spinner[@]} ))
        sleep 0.2
    done
    
    wait $pid
    local status=$?
    local total=$(($(date +%s) - start))
    printf "\r"
    return $status
}

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Marketplace API - Kubernetes Deploy  ${NC}"
echo -e "${BLUE}========================================${NC}"

if $USE_CACHE; then
    echo -e "${CYAN}Mode: Using cached images (--use-cache)${NC}"
else
    echo -e "${CYAN}Mode: Building fresh images${NC}"
fi

TOTAL_START=$(date +%s)

# Check prerequisites
echo -e "\n${YELLOW}Checking prerequisites...${NC}"
command -v docker >/dev/null 2>&1 || { echo -e "${RED}docker is required but not installed.${NC}"; exit 1; }
command -v kind >/dev/null 2>&1 || { echo -e "${RED}kind is required but not installed.${NC}"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo -e "${RED}kubectl is required but not installed.${NC}"; exit 1; }
echo -e "${GREEN}✓ All prerequisites installed${NC}"

# Check if Kind cluster exists
echo -e "\n${YELLOW}Checking Kind cluster...${NC}"
if ! kind get clusters 2>/dev/null | grep -q "kind"; then
    echo -e "${RED}Kind cluster 'kind' not found. Please create it first:${NC}"
    echo -e "  kind create cluster"
    exit 1
fi
echo -e "${GREEN}✓ Kind cluster 'kind' found${NC}"

cd "$PROJECT_DIR"

# Build images (or skip if using cache)
if $USE_CACHE; then
    echo -e "\n${YELLOW}Skipping build, using cached images...${NC}"
    # Verify cached images exist
    if ! docker image inspect keploy-live-record-replay-sandbox-app:latest >/dev/null 2>&1; then
        echo -e "${RED}Cached image 'keploy-live-record-replay-sandbox-app:latest' not found.${NC}"
        echo -e "${YELLOW}Run without --use-cache to build images first.${NC}"
        exit 1
    fi
    if ! docker image inspect keploy-live-record-replay-sandbox-seeder:latest >/dev/null 2>&1; then
        echo -e "${RED}Cached image 'keploy-live-record-replay-sandbox-seeder:latest' not found.${NC}"
        echo -e "${YELLOW}Run without --use-cache to build images first.${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ Cached images found${NC}"
else
    echo -e "\n${YELLOW}Building Docker images...${NC}"
    start_timer
    docker compose build
    echo -e "${GREEN}✓ Images built successfully${NC} $(show_elapsed)"
fi

# Tag images with simple names for Kind
echo -e "\n${YELLOW}Tagging images for Kind...${NC}"
docker tag keploy-live-record-replay-sandbox-app:latest marketplace-api:latest
docker tag keploy-live-record-replay-sandbox-seeder:latest marketplace-seeder:latest
echo -e "${GREEN}✓ Images tagged${NC}"

# Pull helper images needed by init containers (only if not cached)
if $USE_CACHE; then
    echo -e "\n${YELLOW}Checking helper images...${NC}"
    docker image inspect busybox:1.36 >/dev/null 2>&1 || docker pull busybox:1.36
    docker image inspect bitnami/kubectl:latest >/dev/null 2>&1 || docker pull bitnami/kubectl:latest
    echo -e "${GREEN}✓ Helper images ready${NC}"
else
    echo -e "\n${YELLOW}Pulling helper images (busybox, bitnami/kubectl)...${NC}"
    start_timer
    docker pull busybox:1.36 || true
    docker pull bitnami/kubectl:latest || true
    echo -e "${GREEN}✓ Helper images pulled${NC} $(show_elapsed)"
fi

# Load images into Kind cluster
echo -e "\n${YELLOW}Loading images into Kind cluster...${NC}"
start_timer
kind load docker-image marketplace-api:latest
kind load docker-image marketplace-seeder:latest
kind load docker-image busybox:1.36
kind load docker-image bitnami/kubectl:latest
echo -e "${GREEN}✓ Images loaded into Kind${NC} $(show_elapsed)"

# Create namespace if not exists
echo -e "\n${YELLOW}Creating namespace '${NAMESPACE}'...${NC}"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}✓ Namespace ready${NC}"

# Clean up any existing seeder job (to allow re-running)
echo -e "\n${YELLOW}Cleaning up previous deployment...${NC}"
kubectl delete job seeder -n "$NAMESPACE" 2>/dev/null || true
kubectl delete deployment marketplace-api -n "$NAMESPACE" 2>/dev/null || true
echo -e "${GREEN}✓ Cleanup complete${NC}"

# Apply Kubernetes manifests
echo -e "\n${YELLOW}Applying Kubernetes manifests...${NC}"
kubectl apply -k "$PROJECT_DIR/k8s/"
echo -e "${GREEN}✓ Manifests applied${NC}"

# Wait for MySQL to be ready with live timer
echo -e "\n${YELLOW}Waiting for MySQL to be ready...${NC}"
start_timer
kubectl wait --for=condition=ready pod -l app=mysql -n "$NAMESPACE" --timeout=120s &
wait_pid=$!
# Show live timer while waiting
while kill -0 $wait_pid 2>/dev/null; do
    elapsed=$(($(date +%s) - TIMER_START))
    printf "\r${CYAN}  ⏱️  Elapsed: ${elapsed}s${NC}    "
    sleep 1
done
wait $wait_pid
wait_status=$?
printf "\r                              \r"
if [ $wait_status -eq 0 ]; then
    echo -e "${GREEN}✓ MySQL is ready${NC} $(show_elapsed)"
else
    echo -e "${RED}✗ MySQL failed to start${NC}"
    kubectl logs -l app=mysql -n "$NAMESPACE" --tail=20 || true
    exit 1
fi

# Wait for seeder job to complete with live timer
echo -e "\n${YELLOW}Waiting for seeder job to complete...${NC}"
start_timer
kubectl wait --for=condition=complete job/seeder -n "$NAMESPACE" --timeout=180s &
wait_pid=$!
# Show live timer while waiting
while kill -0 $wait_pid 2>/dev/null; do
    elapsed=$(($(date +%s) - TIMER_START))
    printf "\r${CYAN}  ⏱️  Elapsed: ${elapsed}s${NC}    "
    sleep 1
done
wait $wait_pid
wait_status=$?
printf "\r                              \r"
if [ $wait_status -eq 0 ]; then
    echo -e "${GREEN}✓ Seeder completed${NC} $(show_elapsed)"
else
    echo -e "${RED}✗ Seeder failed or timed out${NC}"
    echo -e "${YELLOW}Seeder logs:${NC}"
    kubectl logs job/seeder -n "$NAMESPACE" --tail=30 || true
    echo -e "${YELLOW}Seeder pod status:${NC}"
    kubectl describe pod -l app=seeder -n "$NAMESPACE" | tail -20 || true
    exit 1
fi

# Wait for app to be ready with live timer
echo -e "\n${YELLOW}Waiting for app to be ready...${NC}"
start_timer
kubectl wait --for=condition=ready pod -l app=marketplace-api -n "$NAMESPACE" --timeout=120s &
wait_pid=$!
# Show live timer while waiting
while kill -0 $wait_pid 2>/dev/null; do
    elapsed=$(($(date +%s) - TIMER_START))
    printf "\r${CYAN}  ⏱️  Elapsed: ${elapsed}s${NC}    "
    sleep 1
done
wait $wait_pid
wait_status=$?
printf "\r                              \r"
if [ $wait_status -eq 0 ]; then
    echo -e "${GREEN}✓ App is ready${NC} $(show_elapsed)"
else
    echo -e "${RED}✗ App failed to start${NC}"
    kubectl logs -l app=marketplace-api -n "$NAMESPACE" --tail=20 || true
    exit 1
fi

# Calculate total time
TOTAL_END=$(date +%s)
TOTAL_ELAPSED=$((TOTAL_END - TOTAL_START))

# Show status
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  Deployment Complete!                  ${NC}"
echo -e "${BLUE}  Total time: ${TOTAL_ELAPSED} seconds            ${NC}"
echo -e "${BLUE}========================================${NC}"

echo -e "\n${YELLOW}Pod Status:${NC}"
kubectl get pods -n "$NAMESPACE"

echo -e "\n${YELLOW}Services:${NC}"
kubectl get svc -n "$NAMESPACE"

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}  To access the API, run:${NC}"
echo -e "${GREEN}    kubectl port-forward svc/marketplace-api 1105:1105 -n ${NAMESPACE}${NC}"
echo -e "${GREEN}  Then visit: http://localhost:1105/health${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${YELLOW}To run load tests:${NC}"
echo -e "  python3 scripts/load_test.py --k8s"
echo -e "\n${YELLOW}To clean up:${NC}"
echo -e "  ./scripts/cleanup-k8s.sh"
