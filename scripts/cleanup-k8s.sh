#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

NAMESPACE="live"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Marketplace API - Kubernetes Cleanup ${NC}"
echo -e "${BLUE}========================================${NC}"

# Check if namespace exists
if ! kubectl get namespace "$NAMESPACE" &>/dev/null; then
    echo -e "${YELLOW}Namespace '$NAMESPACE' does not exist. Nothing to clean up.${NC}"
else
    echo -e "\n${YELLOW}Current resources in namespace '$NAMESPACE':${NC}"
    kubectl get all -n "$NAMESPACE" 2>/dev/null || true

    echo ""
    read -p "Delete all resources in namespace '$NAMESPACE'? (y/N) " -n 1 -r
    echo ""

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "\n${YELLOW}Cleaning up Kubernetes resources...${NC}"

        # Delete all resources using kustomize
        echo -e "${YELLOW}Deleting app resources...${NC}"
        kubectl delete -k k8s/ --ignore-not-found=true 2>/dev/null || true

        # Delete any remaining jobs
        echo -e "${YELLOW}Deleting any remaining jobs...${NC}"
        kubectl delete jobs --all -n "$NAMESPACE" --ignore-not-found=true 2>/dev/null || true

        # Delete the namespace
        echo -e "${YELLOW}Deleting namespace '$NAMESPACE'...${NC}"
        kubectl delete namespace "$NAMESPACE" --ignore-not-found=true

        echo -e "${GREEN}✓ Kubernetes resources cleaned up${NC}"
    else
        echo -e "${YELLOW}Skipped Kubernetes cleanup.${NC}"
    fi
fi

# Ask about images in Kind cluster
echo ""
read -p "Also remove app images from Kind cluster? (y/N) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "\n${YELLOW}Removing images from Kind cluster...${NC}"
    
    # Remove images from Kind cluster using crictl
    echo -e "${YELLOW}Removing marketplace-api image from Kind...${NC}"
    docker exec kind-control-plane crictl rmi docker.io/library/marketplace-api:latest 2>/dev/null || true
    
    echo -e "${YELLOW}Removing marketplace-seeder image from Kind...${NC}"
    docker exec kind-control-plane crictl rmi docker.io/library/marketplace-seeder:latest 2>/dev/null || true
    
    echo -e "${GREEN}✓ Images removed from Kind cluster${NC}"
    
    echo -e "\n${YELLOW}Note: Local Docker images are preserved for native/Docker setup.${NC}"
    echo -e "${YELLOW}To delete Kind cluster entirely: kind delete cluster${NC}"
else
    echo -e "${YELLOW}Skipped image cleanup.${NC}"
fi

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}  Cleanup Complete!                     ${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${YELLOW}To redeploy:${NC}"
echo -e "  ./scripts/deploy-k8s.sh"
