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

## Quick Start (Single Command!)

```bash
# Build and start everything - MySQL → Seeder → App (automatic)
# Only shows app logs (hides mysql and seeder logs)
docker compose up --build --attach app
```

This single command will:
1. ✅ Start MySQL and wait for it to be healthy
2. ✅ Run the seeder to populate the database (~45 seconds)
3. ✅ Start the application after seeding completes
4. ✅ Only display app logs (mysql and seeder logs are hidden)

**⏱️ Note**: First run takes ~1-2 minutes as it builds images and seeds ~31MB of test data.

**💡 Tip**: Use `--attach app` to hide mysql/seeder logs. You can always check them later:
```bash
# Check seeder logs
docker compose logs seeder

# Check mysql logs
docker compose logs mysql
```

When ready, the API will be available at `http://localhost:1105`


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

This will automatically:
1. Pull MySQL 8.0 image
2. Build the Go application and seeder images
3. Start MySQL container and wait for it to be healthy
4. Run the seeder to populate the database with test data
5. Start the application after seeding completes successfully

### Step 3: Verify Services

```bash
# Check container status
docker compose ps

# Expected output (after seeding completes):
# NAME                 STATUS                      PORTS
# marketplace-mysql    Up (healthy)                0.0.0.0:3306->3306/tcp
# marketplace-seeder   Exited (0)                  
# marketplace-api      Up                          0.0.0.0:1105->1105/tcp
```

**Note**: The seeder shows "Exited (0)" which means it completed successfully.

### Step 4: Verify Database is Seeded

The seeder runs automatically! You can verify by checking the logs:

```bash
docker compose logs seeder

```

Expected output:
```
🌱 Starting FAST data seeder (Target: ~100MB stored, 300-400MB on query)...
📊 Configuration:
   - Products: 500 (each with ~25KB description, ~15KB attributes)
   - Reviews: ~5000 (each with ~8KB content)
   - Orders: 2000 (each with ~3KB notes)
   - Estimated stored data: ~65MB
   - Estimated seeding time: 3-5 minutes
🗑️  Flushing existing data...
   Cleared table: refresh_tokens
   Cleared table: cart_items
   ...
📊 Seeding dataset...
👥 Creating users...
   Created 56 users
📁 Creating categories...
   Created 22 categories
🛍️ Creating products (with large descriptions)...
   ... created 100/500 products
   ... created 437 products
📍 Creating addresses...
   Created 55 addresses
📦 Creating inventory...
   Created 852 inventory records across 3 warehouses
🎟️ Creating coupons...
   Created 3 coupons
🛒 Creating carts...
   Created cart items
📋 Creating orders (with large notes)...
   ... created 2000 orders
⭐ Creating reviews (with large content)...
   ... created 4000+ reviews

📊 Data Summary:
═══════════════════════════════════════
   users          :     56 records
   categories     :     22 records
   products       :    437 records (~22MB)
   addresses      :     55 records
   inventories    :    852 records
   coupons        :      3 records
   orders         :   2000 records
   order_items    :   9000+ records (~8MB)
   reviews        :   4000+ records
   Total database : ~31MB
═══════════════════════════════════════

✅ Data seeding completed in ~45 seconds!

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
# 1. Start everything (MySQL → Seeder → App - automatic!)
docker compose up -d --build

# 2. Wait for seeding to complete and app to start (~2-3 minutes first time)
docker compose ps

# 3. Test the API
curl http://localhost:1105/health
curl http://localhost:1105/api/v1/products
```

## Docker Compose Commands

### Start Services
```bash
# Start in background (no logs)
docker compose up -d

# Start with only app logs visible (recommended)
docker compose up --attach app

# Start with all logs visible
docker compose up

# Rebuild and start (only app logs)
docker compose up --build --attach app

# Rebuild and start in background
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

# Re-seed database (restart seeder to reset all data)
docker compose up seeder --force-recreate

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
docker compose up seeder --force-recreate

# Or restart everything from scratch
docker compose down && docker compose up -d --build
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
