# Running the E-Commerce Marketplace API - Kubernetes Setup

This guide explains how to deploy and run the application on a Kubernetes cluster using Kind.

## Prerequisites

- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Kind** - [Install Kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- **kubectl** - [Install kubectl](https://kubernetes.io/docs/tasks/tools/)
- **Python 3** - For running load tests

Verify installation:
```bash
docker --version
kind version
kubectl version --client
python3 --version
```

## Quick Start

```bash
# Deploy everything (build, load images, apply manifests)
./scripts/deploy-k8s.sh

# Or use cached images (skip docker compose build)
./scripts/deploy-k8s.sh --use-cache
```

This single command will:
1. ✅ Build Docker images using docker compose (or use cached with `--use-cache`)
2. ✅ Tag and load images into Kind cluster
3. ✅ Create 'live' namespace
4. ✅ Deploy MySQL and wait for it to be ready
5. ✅ Run seeder job to populate database (~45 seconds)
6. ✅ Deploy the application after seeding completes

**⏱️ Note**: First deployment takes ~2-3 minutes. Use `--use-cache` for faster redeployments.

## Step-by-Step Guide

### Step 1: Ensure Kind Cluster Exists

```bash
# Check if Kind cluster exists
kind get clusters

# If not, create one
kind create cluster
```

### Step 2: Build and Load Images

```bash
# Build images using docker compose
docker compose build

# Tag images for Kind
docker tag keploy-live-record-replay-sandbox-app:latest marketplace-api:latest
docker tag keploy-live-record-replay-sandbox-seeder:latest marketplace-seeder:latest

# Load images into Kind
kind load docker-image marketplace-api:latest
kind load docker-image marketplace-seeder:latest
```

### Step 3: Deploy to Kubernetes

```bash
# Create namespace
kubectl create namespace live

# Apply all manifests
kubectl apply -k k8s/

# Or apply individually
kubectl apply -f k8s/mysql.yaml
kubectl apply -f k8s/seeder-job.yaml
kubectl apply -f k8s/app.yaml
```

### Step 4: Wait for Pods to be Ready

```bash
# Watch pod status
kubectl get pods -n live -w

# Wait for MySQL
kubectl wait --for=condition=ready pod -l app=mysql -n live --timeout=120s

# Wait for seeder to complete
kubectl wait --for=condition=complete job/seeder -n live --timeout=180s

# Wait for app
kubectl wait --for=condition=ready pod -l app=marketplace-api -n live --timeout=120s
```

Expected output:
```
NAME                               READY   STATUS      RESTARTS   AGE
marketplace-api-xxxx-xxxx          1/1     Running     0          30s
mysql-xxxx-xxxx                    1/1     Running     0          90s
seeder-xxxx                        0/1     Completed   0          60s
```

### Step 5: Access the Application

```bash
# Port-forward to access the API
kubectl port-forward svc/marketplace-api 1105:1105 -n live
```

In another terminal:
```bash
# Health check
curl http://localhost:1105/health

# Get products
curl http://localhost:1105/api/v1/products
```

## Running Load Tests

The load test script can automatically manage port-forwarding with the `--k8s` flag:

```bash
# Run load test with K8s mode (auto port-forward)
python3 scripts/load_test.py --k8s

# With custom options
python3 scripts/load_test.py --k8s --ops 100 --delay 0.5

# For Native/Docker mode (default, no port-forward)
python3 scripts/load_test.py
```

### Load Test Options

| Option | Description | Default |
|--------|-------------|---------|
| `--k8s` | Enable K8s mode with auto port-forward | Disabled |
| `-n, --namespace` | Kubernetes namespace (with --k8s) | live |
| `-o, --ops` | Number of operations | 250 |
| `-d, --delay` | Delay between operations (seconds) | 2 |
| `-u, --url` | Custom API base URL | localhost:1105 |

## Useful Commands

### View Logs
```bash
# App logs
kubectl logs -f deployment/marketplace-api -n live

# MySQL logs
kubectl logs -f deployment/mysql -n live

# Seeder logs
kubectl logs job/seeder -n live
```

### Check Status
```bash
# All resources in live namespace
kubectl get all -n live

# Pod status
kubectl get pods -n live

# Services
kubectl get svc -n live
```

### Re-seed Database

To reset the database and re-run the seeder:

```bash
# Delete existing seeder job
kubectl delete job seeder -n live

# Re-apply seeder job
kubectl apply -f k8s/seeder-job.yaml

# Wait for completion
kubectl wait --for=condition=complete job/seeder -n live --timeout=180s

# Restart app to pick up fresh data
kubectl rollout restart deployment/marketplace-api -n live
```

### Clean Up

```bash
# Use the cleanup script (recommended)
./scripts/cleanup-k8s.sh

# Or manually delete resources:
# Delete just the app resources
kubectl delete -k k8s/

# Delete the entire namespace
kubectl delete namespace live

# Delete Kind cluster (if needed)
kind delete cluster
```

## Troubleshooting

### 1. Images Not Found

```bash
# Verify images are loaded
docker exec -it kind-control-plane crictl images | grep marketplace

# Reload images if needed
kind load docker-image marketplace-api:latest
kind load docker-image marketplace-seeder:latest
```

### 2. MySQL Not Starting

```bash
# Check MySQL logs
kubectl logs deployment/mysql -n live

# Check events
kubectl describe pod -l app=mysql -n live
```

### 3. Seeder Failing

```bash
# Check seeder logs
kubectl logs job/seeder -n live

# Check if MySQL is accessible
kubectl exec -it deployment/mysql -n live -- mysql -u marketplace -pmarketplace123 -e "SHOW DATABASES;"
```

### 4. App Not Starting

```bash
# Check init container logs (waiting for seeder)
kubectl logs deployment/marketplace-api -n live -c wait-for-seeder

# Check app logs
kubectl logs deployment/marketplace-api -n live -c marketplace-api

# Describe pod for events
kubectl describe pod -l app=marketplace-api -n live
```

### 5. Port-Forward Not Working

```bash
# Check if service exists
kubectl get svc marketplace-api -n live

# Check if pods are ready
kubectl get pods -l app=marketplace-api -n live

# Try different port
kubectl port-forward svc/marketplace-api 8080:1105 -n live
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Kind Cluster                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  Namespace: live                       │  │
│  │                                                        │  │
│  │  ┌──────────┐    ┌──────────┐    ┌──────────────────┐ │  │
│  │  │  MySQL   │◄───│  Seeder  │    │  marketplace-api │ │  │
│  │  │  (Pod)   │    │  (Job)   │    │   (Deployment)   │ │  │
│  │  └────▲─────┘    └──────────┘    └────────▲─────────┘ │  │
│  │       │                                    │           │  │
│  │  ┌────┴─────┐                        ┌────┴─────┐     │  │
│  │  │ Service  │                        │ Service  │     │  │
│  │  │  mysql   │                        │ api:1105 │     │  │
│  │  └──────────┘                        └────▲─────┘     │  │
│  │                                           │           │  │
│  └───────────────────────────────────────────┼───────────┘  │
│                                              │              │
└──────────────────────────────────────────────┼──────────────┘
                                               │
                                    kubectl port-forward
                                               │
                                               ▼
                                    http://localhost:1105
```

## Deployment Flow

1. **MySQL Deployment** starts first
2. **Seeder Job** runs after MySQL is healthy (init container waits for MySQL)
3. **App Deployment** starts after Seeder job completes (init container waits for seeder)

## Test Credentials (After Seeding)

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@marketplace.com | Password123! |
| Seller | seller1@marketplace.com | Password123! |
| Customer | john.doe@email.com | Password123! |

## Next Steps

- Check `flow.md` for API flow diagrams
- See the `openapi.yaml` for complete API documentation
- Import `postman_collection.json` into Postman for testing
