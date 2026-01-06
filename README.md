# E-Commerce Marketplace API

A full-featured RESTful API for an e-commerce marketplace built with **Go** and **MySQL**. This project serves as a sandbox for testing API recording and replay tools like [Keploy](https://keploy.io).

## 🚀 Features

- **User Management** - Registration, authentication (JWT), roles (Admin/Seller/Customer)
- **Product Catalog** - CRUD operations, categories, search, filtering
- **Shopping Cart** - Add/remove items, quantity management
- **Order Processing** - Checkout, order history, status tracking
- **Inventory Management** - Stock tracking, low-stock alerts
- **Reviews & Ratings** - Product reviews with ratings
- **Coupons & Discounts** - Promo codes, percentage/fixed discounts
- **Analytics Dashboard** - Sales reports, bestsellers, customer insights

## 📊 API Endpoints

| Resource | Endpoints |
|----------|-----------|
| Auth | `/api/v1/auth/register`, `/login`, `/me`, `/refresh` |
| Users | `/api/v1/users` - CRUD operations |
| Products | `/api/v1/products` - Search, filter, paginate |
| Categories | `/api/v1/categories` - Category tree |
| Cart | `/api/v1/cart` - Add, remove, validate |
| Orders | `/api/v1/orders` - Checkout, history |
| Coupons | `/api/v1/coupons` - Create, validate |
| Inventory | `/api/v1/inventory` - Stock management |
| Analytics | `/api/v1/analytics` - Sales, products, customers |

## 🏃 Quick Start

Choose your preferred deployment method:

| Method | Command | Documentation |
|--------|---------|---------------|
| **Native** | `go run cmd/server/main.go` | [Native.Readme.md](Native.Readme.md) |
| **Docker** | `docker compose up --attach app` | [Docker.Readme.md](Docker.Readme.md) |
| **Kubernetes** | `./scripts/deploy-k8s.sh` | [K8s.Readme.md](K8s.Readme.md) |

### Docker (Fastest)

```bash
# Start everything (MySQL → Seeder → App)
docker compose up --build --attach app

# API available at http://localhost:1105
curl http://localhost:1105/health
```

### Native

```bash
# Start MySQL
docker run -d --name marketplace-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpassword \
  -e MYSQL_DATABASE=marketplace_db \
  -e MYSQL_USER=marketplace \
  -e MYSQL_PASSWORD=marketplace123 \
  -p 3306:3306 mysql:8.0

# Setup environment
cp .env.example .env

# Seed database
go run cmd/seeder/main.go

# Start server
go run cmd/server/main.go
```

### Kubernetes (Kind)

```bash
# Deploy to Kind cluster
./scripts/deploy-k8s.sh

# Port-forward and test
kubectl port-forward svc/marketplace-api 1105:1105 -n live
curl http://localhost:1105/health
```

## 🧪 Load Testing

The project includes a Python load testing script that performs 250+ API operations:

```bash
# Native/Docker mode (default)
python3 scripts/load_test.py

# Kubernetes mode (auto port-forward)
python3 scripts/load_test.py --k8s

# Custom options
python3 scripts/load_test.py --ops 100 --delay 0.5
```

## 🔑 Test Credentials

After seeding, use these credentials:

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@marketplace.com | Password123! |
| Seller | seller1@marketplace.com | Password123! |
| Customer | john.doe@email.com | Password123! |

## 📁 Project Structure

```
.
├── cmd/
│   ├── server/         # Main API server
│   └── seeder/         # Database seeder
├── internal/
│   ├── auth/           # JWT authentication
│   ├── config/         # Configuration
│   ├── database/       # Database connection
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # Auth, logging middleware
│   ├── models/         # Data models
│   └── routes/         # Route definitions
├── k8s/                # Kubernetes manifests
├── scripts/
│   ├── deploy-k8s.sh   # K8s deployment script
│   ├── cleanup-k8s.sh  # K8s cleanup script
│   ├── cleanup.sh      # Docker cleanup script
│   └── load_test.py    # Load testing script
├── Dockerfile          # App container
├── Dockerfile.seeder   # Seeder container
├── docker-compose.yml  # Docker Compose config
├── openapi.yaml        # API specification
└── postman_collection.json
```

## 📚 Documentation

- **[Native.Readme.md](Native.Readme.md)** - Running natively with Go
- **[Docker.Readme.md](Docker.Readme.md)** - Running with Docker Compose
- **[K8s.Readme.md](K8s.Readme.md)** - Running on Kubernetes
- **[openapi.yaml](openapi.yaml)** - OpenAPI 3.0 specification
- **[postman_collection.json](postman_collection.json)** - Postman collection

## 🛠️ Tech Stack

- **Language**: Go 1.25
- **Database**: MySQL 8.0
- **Authentication**: JWT (RS256)
- **ORM**: GORM
- **Container**: Docker, Kubernetes
- **Testing**: Python (load tests)

## 🧹 Cleanup

```bash
# Docker cleanup
./scripts/cleanup.sh

# Kubernetes cleanup
./scripts/cleanup-k8s.sh
```

## 📄 License

MIT License - feel free to use this project for testing and learning purposes.

---

Made with ❤️ for testing [Keploy](https://keploy.io) - API Testing Made Simple
