// for large size mocks
// Package handlers - Heavy data handlers for testing large database operations
// These endpoints are NOT part of regular application flow - used for mock testing purposes
package handlers

import (
	"net/http"
	"time"
	"unsafe"

	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

// for large size mocks
// HeavyHandler handles heavy data operations for testing large database pulls
type HeavyHandler struct {
	db *database.Database
}

// for large size mocks
// NewHeavyHandler creates a new heavy data handler
func NewHeavyHandler(db *database.Database) *HeavyHandler {
	return &HeavyHandler{db: db}
}

// for large size mocks
// HeavyResponse is the standard response format for heavy endpoints
type HeavyResponse struct {
	Operation       string   `json:"operation"`
	DataPulledBytes int64    `json:"dataPulledBytes"`
	DataPulledMB    float64  `json:"dataPulledMB"`
	RecordCount     int      `json:"recordCount"`
	TablesAccessed  []string `json:"tablesAccessed"`
	QueryTimeMs     int64    `json:"queryTimeMs"`
	Message         string   `json:"message"`
}

// for large size mocks
// estimateSize estimates the size of data in bytes
func estimateSize(data interface{}) int64 {
	return int64(unsafe.Sizeof(data))
}

// for large size mocks
// calculateStringSize calculates approximate size of string data
func calculateStringSize(s string) int64 {
	return int64(len(s))
}

// for large size mocks
// HeavyProducts fetches all products with full descriptions and attributes
func (h *HeavyHandler) HeavyProducts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Preload("Inventory").Find(&products)

	// Calculate data size
	var totalSize int64
	for _, p := range products {
		totalSize += calculateStringSize(p.Description)
		totalSize += calculateStringSize(p.Attributes)
		totalSize += calculateStringSize(p.Images)
		totalSize += calculateStringSize(p.Name)
		totalSize += calculateStringSize(p.SKU)
		totalSize += 200 // overhead for other fields
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_products",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(products),
		TablesAccessed:  []string{"products", "categories", "users", "reviews", "inventories"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy products operation completed - fetched all products with relations",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyOrders fetches all orders with items, payments, and addresses
func (h *HeavyHandler) HeavyOrders(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var orders []models.Order
	h.db.Preload("Items").Preload("Items.Product").Preload("Payment").Preload("ShippingAddress").Preload("User").Find(&orders)

	var totalSize int64
	for _, o := range orders {
		totalSize += calculateStringSize(o.Notes)
		totalSize += calculateStringSize(o.OrderNumber)
		for _, item := range o.Items {
			totalSize += calculateStringSize(item.ProductName)
			totalSize += calculateStringSize(item.SKU)
			totalSize += 100
		}
		totalSize += 500 // overhead
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_orders",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(orders),
		TablesAccessed:  []string{"orders", "order_items", "products", "payments", "addresses", "users"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy orders operation completed - fetched all orders with relations",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyReviews fetches all reviews with user and product relations
func (h *HeavyHandler) HeavyReviews(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var reviews []models.Review
	h.db.Preload("User").Preload("Product").Find(&reviews)

	var totalSize int64
	for _, rev := range reviews {
		totalSize += calculateStringSize(rev.Content)
		totalSize += calculateStringSize(rev.Title)
		totalSize += calculateStringSize(rev.Images)
		totalSize += 200
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_reviews",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(reviews),
		TablesAccessed:  []string{"reviews", "users", "products"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy reviews operation completed - fetched all reviews with relations",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyInventory fetches all inventory with product details
func (h *HeavyHandler) HeavyInventory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var inventory []models.Inventory
	h.db.Preload("Product").Find(&inventory)

	var totalSize int64
	for _, inv := range inventory {
		totalSize += calculateStringSize(inv.WarehouseName)
		totalSize += 150
	}

	// Also fetch products for full data
	var products []models.Product
	h.db.Find(&products)
	for _, p := range products {
		totalSize += calculateStringSize(p.Description)
		totalSize += calculateStringSize(p.Attributes)
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_inventory",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(inventory),
		TablesAccessed:  []string{"inventories", "products"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy inventory operation completed - fetched all inventory with products",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyAggregate performs cross-table aggregation from all tables
func (h *HeavyHandler) HeavyAggregate(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var totalSize int64
	var totalRecords int

	// Products with all relations
	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Preload("Inventory").Find(&products)
	for _, p := range products {
		totalSize += calculateStringSize(p.Description) + calculateStringSize(p.Attributes)
	}
	totalRecords += len(products)

	// Orders with all relations
	var orders []models.Order
	h.db.Preload("Items").Preload("Payment").Find(&orders)
	for _, o := range orders {
		totalSize += calculateStringSize(o.Notes)
	}
	totalRecords += len(orders)

	// Reviews
	var reviews []models.Review
	h.db.Find(&reviews)
	for _, r := range reviews {
		totalSize += calculateStringSize(r.Content)
	}
	totalRecords += len(reviews)

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_aggregate",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     totalRecords,
		TablesAccessed:  []string{"products", "categories", "users", "reviews", "inventories", "orders", "order_items", "payments"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy aggregate operation completed - cross-table aggregation",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyUsers fetches all users with addresses, orders, and reviews
func (h *HeavyHandler) HeavyUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var users []models.User
	h.db.Preload("Addresses").Preload("Orders").Preload("Orders.Items").Preload("Reviews").Find(&users)

	var totalSize int64
	for _, u := range users {
		totalSize += calculateStringSize(u.Email)
		totalSize += calculateStringSize(u.FirstName)
		totalSize += calculateStringSize(u.LastName)
		for _, addr := range u.Addresses {
			totalSize += calculateStringSize(addr.AddressLine1)
			totalSize += calculateStringSize(addr.City)
			totalSize += 100
		}
		for _, order := range u.Orders {
			totalSize += calculateStringSize(order.Notes)
		}
		totalSize += 300
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_users",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(users),
		TablesAccessed:  []string{"users", "addresses", "orders", "order_items", "reviews"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy users operation completed - fetched all users with relations",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyCategories fetches all categories with their products
func (h *HeavyHandler) HeavyCategories(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var categories []models.Category
	h.db.Preload("Products").Preload("Products.Reviews").Preload("Children").Find(&categories)

	var totalSize int64
	for _, c := range categories {
		totalSize += calculateStringSize(c.Description)
		totalSize += calculateStringSize(c.Name)
		for _, p := range c.Products {
			totalSize += calculateStringSize(p.Description)
			totalSize += calculateStringSize(p.Attributes)
		}
		totalSize += 200
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_categories",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(categories),
		TablesAccessed:  []string{"categories", "products", "reviews"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy categories operation completed - fetched all categories with products",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyPayments fetches all payments with order details
func (h *HeavyHandler) HeavyPayments(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var payments []models.Payment
	h.db.Find(&payments)

	// Also fetch related orders
	var orders []models.Order
	h.db.Preload("Items").Preload("User").Find(&orders)

	var totalSize int64
	for _, p := range payments {
		totalSize += calculateStringSize(p.GatewayResponse)
		totalSize += calculateStringSize(p.TransactionID)
		totalSize += 200
	}
	for _, o := range orders {
		totalSize += calculateStringSize(o.Notes)
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_payments",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(payments),
		TablesAccessed:  []string{"payments", "orders", "order_items", "users"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy payments operation completed - fetched all payments with orders",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyCarts fetches all carts with items and products
func (h *HeavyHandler) HeavyCarts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var carts []models.Cart
	h.db.Preload("Items").Preload("Items.Product").Find(&carts)

	var totalSize int64
	for _, c := range carts {
		for _, item := range c.Items {
			totalSize += calculateStringSize(item.Product.Description)
			totalSize += calculateStringSize(item.Product.Attributes)
			totalSize += 100
		}
		totalSize += 50
	}

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_carts",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     len(carts),
		TablesAccessed:  []string{"carts", "cart_items", "products"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy carts operation completed - fetched all carts with items",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyFullDump fetches all data from all tables - full database dump
func (h *HeavyHandler) HeavyFullDump(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var totalSize int64
	var totalRecords int

	// Users with all relations
	var users []models.User
	h.db.Preload("Addresses").Preload("Orders").Preload("Reviews").Find(&users)
	for _, u := range users {
		totalSize += calculateStringSize(u.Email) + calculateStringSize(u.FirstName) + 100
	}
	totalRecords += len(users)

	// Products with all relations
	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Preload("Inventory").Find(&products)
	for _, p := range products {
		totalSize += calculateStringSize(p.Description) + calculateStringSize(p.Attributes) + calculateStringSize(p.Images)
	}
	totalRecords += len(products)

	// Orders with all relations
	var orders []models.Order
	h.db.Preload("Items").Preload("Payment").Preload("ShippingAddress").Find(&orders)
	for _, o := range orders {
		totalSize += calculateStringSize(o.Notes)
	}
	totalRecords += len(orders)

	// Reviews
	var reviews []models.Review
	h.db.Find(&reviews)
	for _, rev := range reviews {
		totalSize += calculateStringSize(rev.Content) + calculateStringSize(rev.Title)
	}
	totalRecords += len(reviews)

	// Categories
	var categories []models.Category
	h.db.Find(&categories)
	for _, c := range categories {
		totalSize += calculateStringSize(c.Description)
	}
	totalRecords += len(categories)

	// Inventory
	var inventory []models.Inventory
	h.db.Find(&inventory)
	totalRecords += len(inventory)
	totalSize += int64(len(inventory) * 100)

	// Payments
	var payments []models.Payment
	h.db.Find(&payments)
	for _, p := range payments {
		totalSize += calculateStringSize(p.GatewayResponse)
	}
	totalRecords += len(payments)

	// Carts
	var carts []models.Cart
	h.db.Preload("Items").Find(&carts)
	totalRecords += len(carts)
	totalSize += int64(len(carts) * 50)

	elapsed := time.Since(start).Milliseconds()

	response := HeavyResponse{
		Operation:       "heavy_full_dump",
		DataPulledBytes: totalSize,
		DataPulledMB:    float64(totalSize) / 1024 / 1024,
		RecordCount:     totalRecords,
		TablesAccessed:  []string{"users", "addresses", "products", "categories", "orders", "order_items", "payments", "reviews", "inventories", "carts", "cart_items"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy full dump operation completed - fetched all data from all tables",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}
