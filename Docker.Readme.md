# Running the E-Commerce Marketplace API - Docker Setup

This guide explains how to run the application using Docker and Docker Compose.

## Prerequisites

- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose** - Usually included with Docker Desktop

Verify installation:
```bash
docker --version
docker compose --version
```

## Quick Start (3 Commands)

```bash
# 1. Start MySQL and Application
docker compose up -d

# 2. Wait for services to be healthy (about 30-60 seconds)
docker compose ps

# 3. Seed the database with test data (run once)
docker compose --profile seed run --rm seeder
```

That's it! The API will be available at `http://localhost:1105`


## Step-by-Step Guide

### Step 1: Clone the Repository

```bash
cd /path/to/keploy-live-record-replay-sandbox
```

### Step 2: Build and Start Services

```bash
# Build images and start containers in detached mode
docker compose up -d --build
```

This will:
1. Pull MySQL 8.0 image
2. Build the Go application image
3. Start MySQL container
4. Wait for MySQL to be healthy
5. Start the application container

### Step 3: Verify Services

```bash
# Check container status
docker compose ps

# Expected output:
# NAME                 STATUS                   PORTS
# marketplace-mysql    Up (healthy)             0.0.0.0:3306->3306/tcp
# marketplace-api      Up                       0.0.0.0:1105->1105/tcp
```

### Step 4: Seed the Database (REQUIRED)

**⚠️ Important**: This step must be run BEFORE using the API. It will:
- Flush/clear ALL existing data from tables
- Populate all tables with fresh test data (mix of small and large datasets)

```bash
# Run the data seeder inside the container
docker compose --profile seed run --rm seeder
```

Expected output:
```
🌱 Starting data seeder...
🗑️  Flushing existing data...
   Cleared table: refresh_tokens
   Cleared table: cart_items
   ...
📊 Seeding fresh data...
👥 Creating users...
   Created 26 users
📁 Creating categories...
   Created 28 categories (8 parent + 20 subcategories)
🛍️ Creating products...
   Created 56 products
📍 Creating addresses...
   Created 13 addresses
📦 Creating inventory...
   Created 100+ inventory records across 3 warehouses
🎟️ Creating coupons...
   Created 8 coupons
🛒 Creating carts with items...
   Added items to 7 carts
📋 Creating orders...
   Created 50+ orders with items and payments
⭐ Creating reviews...
   Created 100+ reviews

📊 Data Summary:
═══════════════════════════════════════
   users          : 26 records
   categories     : 28 records
   products       : 56 records
   addresses      : 13 records
   inventories    : 100+ records
   coupons        : 8 records
   orders         : 50+ records
   reviews        : 100+ records
═══════════════════════════════════════

🔑 Test Credentials:
   Admin:    admin@marketplace.com / Password123!
   Seller:   seller1@marketplace.com / Password123!
   Customer: john.doe@email.com / Password123!
```

### Step 5: Check Application Health

```bash
curl http://localhost:1105/health
```

Expected response:
```json
{"status":"healthy","database":"connected"}
```

## Complete Workflow Summary

```bash
# 1. Start containers
docker compose up -d --build

# 2. Wait for healthy status
docker compose ps

# 3. Seed database (flushes old data, adds fresh data)
docker compose --profile seed run --rm seeder

# 4. Test the API
curl http://localhost:1105/health
curl http://localhost:1105/api/v1/products
```

## Docker Compose Commands

### Start Services
```bash
# Start in background
docker compose up -d

# Start with logs visible
docker compose up

# Rebuild and start
docker compose up -d --build
```

### Stop Services
```bash
# Stop containers (keeps data)
docker compose stop

# Stop and remove containers (keeps data in volumes)
docker compose down

# Stop and remove everything including volumes (DELETES ALL DATA)
docker compose down -v
```

### View Logs
```bash
# All services
docker compose logs -f

# Application only
docker compose logs -f app

# MySQL only
docker compose logs -f mysql
```

### Execute Commands
```bash
# Open shell in app container
docker compose exec app sh

# Re-seed database (reset all data)
docker compose --profile seed run --rm seeder

# Connect to MySQL
docker compose exec mysql mysql -u marketplace -pmarketplace123 marketplace_db

# Run MySQL queries
docker compose exec mysql mysql -u marketplace -pmarketplace123 -e "SHOW TABLES;" marketplace_db
```

### Restart Services
```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart app
```

## Environment Configuration

The Docker Compose file uses these default environment variables:

| Variable | Default Value | Description |
|----------|---------------|-------------|
| `PORT` | 1105 | Application port |
| `ENV` | development | Environment mode |
| `DB_HOST` | mysql | MySQL host (container name) |
| `DB_PORT` | 3306 | MySQL port |
| `DB_USER` | marketplace | MySQL user |
| `DB_PASSWORD` | marketplace123 | MySQL password |
| `DB_NAME` | marketplace_db | Database name |
| `JWT_SECRET` | your-super-secret... | JWT signing key |
| `JWT_EXPIRY_HOURS` | 24 | Access token expiry |
| `JWT_REFRESH_EXPIRY_HOURS` | 168 | Refresh token expiry |

### Override Environment Variables

Create a `.env` file in the project root:

```bash
# .env
JWT_SECRET=my-production-secret
ENV=production
```

Or use environment variables directly:

```bash
JWT_SECRET=my-secret docker compose up -d
```

## Data Persistence

MySQL data is persisted in a Docker volume:
- Volume name: `mysql_data`
- Data survives container restarts
- Data is deleted only with `docker compose down -v`

### Backup Database
```bash
docker compose exec mysql mysqldump -u marketplace -pmarketplace123 marketplace_db > backup.sql
```

### Restore Database
```bash
docker compose exec -T mysql mysql -u marketplace -pmarketplace123 marketplace_db < backup.sql
```

### Reset All Data
```bash
# Re-run the seeder to flush and repopulate
docker compose --profile seed run --rm seeder
```

## Troubleshooting

### 1. Port Already in Use
```bash
# Check what's using port 1105
lsof -i :1105

# Use different port
PORT=3000 docker compose up -d
```

### 2. MySQL Not Ready
```bash
# Wait for MySQL to be fully ready
docker compose logs mysql

# Restart the app after MySQL is healthy
docker compose restart app
```

### 3. Build Failures
```bash
# Remove old images and rebuild
docker compose down
docker compose build --no-cache
docker compose up -d
```

### 4. Container Exiting Immediately
```bash
# Check logs for error
docker compose logs app

# Common fix: wait for MySQL
docker compose restart app
```

### 5. Clean Start
```bash
# Remove everything and start fresh
docker compose down -v
docker system prune -f
docker compose up -d --build
docker compose --profile seed run --rm seeder
```

## Testing the API

### Using cURL
```bash
# Health check
curl http://localhost:1105/health

# Login with seeded user
curl -X POST http://localhost:1105/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john.doe@email.com","password":"Password123!"}'

# Get products
curl http://localhost:1105/api/v1/products
```

### Using Postman
Import the `postman_collection.json` file into Postman for a complete API testing experience with:
- Auto-token management
- All endpoints organized by category
- Clear dependency order for API calls

## Test Credentials (After Seeding)

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@marketplace.com | Password123! |
| Seller | seller1@marketplace.com | Password123! |
| Seller | seller2@marketplace.com | Password123! |
| Customer | john.doe@email.com | Password123! |
| Customer | jane.smith@email.com | Password123! |

## Production Considerations

For production deployment:

1. **Change JWT Secret**: Use a strong, random secret
2. **Use Secrets Manager**: Don't hardcode passwords
3. **Enable TLS/SSL**: Use HTTPS
4. **Resource Limits**: Add CPU/memory limits in docker-compose
5. **Logging**: Configure proper log aggregation
6. **Monitoring**: Add health monitoring

Example production docker-compose override:

```yaml
# docker-compose.prod.yml
services:
  app:
    environment:
      - ENV=production
      - JWT_SECRET=${JWT_SECRET}
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
```

Run with:
```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## Next Steps

- Check `flow.md` for API flow diagrams
- See the `openapi.yaml` for complete API documentation
- Import `postman_collection.json` into Postman for testing
