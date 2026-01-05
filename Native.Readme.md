# Running the E-Commerce Marketplace API - Native Setup

This guide explains how to run the Go application natively while using Docker for MySQL.

## Prerequisites

- **Go 1.25+** - [Download Go](https://go.dev/dl/)
- **Docker** - [Install Docker](https://docs.docker.com/get-docker/)
- **Git** - [Download Git](https://git-scm.com/downloads)

## Step 1: Start MySQL with Docker

Run MySQL in a Docker container:

```bash
docker run -d \
  --name marketplace-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=marketplace_db \
  -e MYSQL_USER=marketplace \
  -e MYSQL_PASSWORD=marketplace123 \
  -p 3306:3306 \
  mysql:8.0 --default-authentication-plugin=mysql_native_password
```

Wait for MySQL to be ready (about 30 seconds):
```bash
# Check if MySQL is ready
docker logs marketplace-mysql 2>&1 | grep "ready for connections"

# Or wait and check status
sleep 30 && docker ps
```

## Step 2: Clone and Setup Application

```bash
# Navigate to the project directory
cd /path/to/keploy-live-record-replay-sandbox

# Install Go dependencies
go mod download

# Or if go.sum doesn't exist
go mod tidy
```

## Step 3: Configure Environment Variables

```bash
# Copy the example environment file
cp .env.example .env
```

The default `.env` configuration works with Docker MySQL:
```env
PORT=1105
ENV=development
DB_HOST=localhost
DB_PORT=3306
DB_USER=marketplace
DB_PASSWORD=marketplace123
DB_NAME=marketplace_db
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_EXPIRY_HOURS=24
JWT_REFRESH_EXPIRY_HOURS=168
```

## Step 4: Seed the Database (REQUIRED)

**⚠️ Important**: Run this BEFORE starting the application:

```bash
go run cmd/seeder/main.go
```

Expected output:
```
🌱 Starting data seeder...
🗑️  Flushing existing data...
📊 Seeding fresh data...
👥 Creating users...
   Created 26 users
📁 Creating categories...
   Created 28 categories
🛍️ Creating products...
   Created 54 products
...
✅ Data seeding completed successfully!

🔑 Test Credentials:
   Admin:    admin@marketplace.com / Password123!
   Seller:   seller1@marketplace.com / Password123!
   Customer: john.doe@email.com / Password123!
```

## Step 5: Run the Application

```bash
# Run the application
go run cmd/server/main.go

# Or build and run
go build -o main ./cmd/server
./main
```

## Step 6: Verify the Setup

Check if the server is running:
```bash
curl http://localhost:1105/health
```

Expected response:
```json
{"status":"healthy","database":"connected"}
```

## Complete Workflow Summary

```bash
# 1. Start MySQL container
docker run -d --name marketplace-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=marketplace_db \
  -e MYSQL_USER=marketplace \
  -e MYSQL_PASSWORD=marketplace123 \
  -p 3306:3306 \
  mysql:8.0 --default-authentication-plugin=mysql_native_password

# 2. Wait for MySQL to be ready
sleep 30

# 3. Setup environment
cp .env.example .env

# 4. Seed database
go run cmd/seeder/main.go

# 5. Start server
go run cmd/server/main.go
```

## Quick Start (One Command)

After MySQL is running:
```bash
cp .env.example .env && go run cmd/seeder/main.go && go run cmd/server/main.go
```

## Managing MySQL Container

```bash
# Stop MySQL
docker stop marketplace-mysql

# Start MySQL (after stopping)
docker start marketplace-mysql

# View MySQL logs
docker logs marketplace-mysql

# Connect to MySQL CLI
docker exec -it marketplace-mysql mysql -u marketplace -pmarketplace123 marketplace_db

# Remove MySQL container (deletes data)
docker rm -f marketplace-mysql
```

## Re-seeding Data

To reset all data and start fresh:
```bash
# Stop the Go application (Ctrl+C), then:
go run cmd/seeder/main.go

# Start server again
go run cmd/server/main.go
```

## Common Issues

### 1. MySQL Connection Refused
- Ensure MySQL container is running: `docker ps | grep marketplace-mysql`
- Wait for MySQL to initialize: `docker logs marketplace-mysql`
- Check if port 3306 is available: `netstat -tlnp | grep 3306`

### 2. Port 3306 Already in Use
```bash
# Find what's using the port
lsof -i :3306

# Use a different port
docker run -d --name marketplace-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=marketplace_db \
  -e MYSQL_USER=marketplace \
  -e MYSQL_PASSWORD=marketplace123 \
  -p 3307:3306 \
  mysql:8.0 --default-authentication-plugin=mysql_native_password

# Update .env file
DB_PORT=3307
```

### 3. Module Not Found
```bash
go mod tidy
```

### 4. Permission Denied
```bash
chmod +x ./main
```

## Testing the API

### Using cURL
```bash
# Login with seeded user
curl -X POST http://localhost:1105/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john.doe@email.com","password":"Password123!"}'

# Get products
curl http://localhost:1105/api/v1/products
```

### Using Postman
Import the `postman_collection.json` file into Postman for a complete API testing experience.

## Test Credentials (After Seeding)

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@marketplace.com | Password123! |
| Seller | seller1@marketplace.com | Password123! |
| Customer | john.doe@email.com | Password123! |
| Customer | jane.smith@email.com | Password123! |

## Cleanup

To remove everything and start fresh:
```bash
# Stop the Go application (Ctrl+C)

# Remove MySQL container
docker rm -f marketplace-mysql

# Remove local files
rm -f ./main ./.env

# Or use the cleanup script
./scripts/cleanup.sh
```

## Next Steps

- Check `flow.md` for API flow diagrams
- See the `openapi.yaml` for complete API documentation
- Import `postman_collection.json` into Postman for testing
