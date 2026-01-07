# Heavy Data API - Large Database Operations

> ⚠️ **NOT part of regular application flow** - These endpoints are designed specifically for testing large mock scenarios (e.g., Keploy mock testing with large responses).

## Purpose

The `/api/v1/heavy/*` endpoints execute heavy database queries (~40MB+ per call) but return only lightweight metadata. This allows testing scenarios where the application pulls large amounts of data from the database.

## Endpoints

| Endpoint | Description | Est. Data Pull |
|----------|-------------|----------------|
| `GET /api/v1/heavy/products` | All products with relations | ~20MB |
| `GET /api/v1/heavy/orders` | All orders with items/payments | ~35MB |
| `GET /api/v1/heavy/reviews` | All reviews with user/product | ~35MB |
| `GET /api/v1/heavy/inventory` | All inventory with products | ~15MB |
| `GET /api/v1/heavy/aggregate` | Cross-table aggregation | ~40MB+ |
| `GET /api/v1/heavy/users` | All users with addresses/orders | ~30MB |
| `GET /api/v1/heavy/categories` | Categories with products | ~25MB |
| `GET /api/v1/heavy/payments` | Payments with orders | ~25MB |
| `GET /api/v1/heavy/carts` | Carts with items/products | ~20MB |
| `GET /api/v1/heavy/full-dump` | Full database dump | ~50MB+ |

## Response Format

```json
{
  "operation": "heavy_products",
  "dataPulledBytes": 20971520,
  "dataPulledMB": 20.0,
  "recordCount": 500,
  "tablesAccessed": ["products", "categories", "users"],
  "queryTimeMs": 1234,
  "message": "Heavy data operation completed"
}
```

---

## Usage

### Prerequisites

1. Seed the database with test data:
   ```bash
   go run cmd/seeder/main.go
   ```

### Native Environment

1. Start the server:
   ```bash
   go run cmd/server/main.go
   ```

2. Run heavy load test:
   ```bash
   # Test 5 APIs (default)
   python3 scripts/heavy_load_test.py
   
   # Test all 10 APIs
   python3 scripts/heavy_load_test.py --api 10
   
   # Test specific number of APIs
   python3 scripts/heavy_load_test.py --api 3
   ```

### Docker Environment

1. Start with docker-compose:
   ```bash
   docker-compose up -d
   ```

2. Run heavy load test:
   ```bash
   python3 scripts/heavy_load_test.py --api 10
   ```

### Kubernetes Environment

1. Deploy to Kubernetes:
   ```bash
   ./scripts/deploy-k8s.sh
   ```

2. Port-forward the service:
   ```bash
   kubectl port-forward svc/marketplace-api 1105:1105
   ```

3. Run heavy load test with `--k8s` flag:
   ```bash
   # Test 5 APIs on k8s
   python3 scripts/heavy_load_test.py --k8s
   
   # Test all 10 APIs on k8s
   python3 scripts/heavy_load_test.py --api 10 --k8s
   ```

---

## Code Identification

All code for this feature is marked with:
```
// for large size mocks
```

Files:
- `internal/handlers/heavy.go` - Handler implementation
- `internal/router/router.go` - Route registration (search for "heavy")
- `scripts/heavy_load_test.py` - Python test script
