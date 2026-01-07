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

	// Fetch data from database (for actual DB interaction)
	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Preload("Inventory").Find(&products)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 41943040 // 40 MB
	const fixedRecords = 5000

	response := HeavyResponse{
		Operation:       "heavy_products",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var orders []models.Order
	h.db.Preload("Items").Preload("Items.Product").Preload("Payment").Preload("ShippingAddress").Preload("User").Find(&orders)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 44040192 // 42 MB
	const fixedRecords = 8000

	response := HeavyResponse{
		Operation:       "heavy_orders",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var reviews []models.Review
	h.db.Preload("User").Preload("Product").Find(&reviews)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 39845888 // 38 MB
	const fixedRecords = 12000

	response := HeavyResponse{
		Operation:       "heavy_reviews",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var inventory []models.Inventory
	h.db.Preload("Product").Find(&inventory)

	var products []models.Product
	h.db.Find(&products)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 36700160 // 35 MB
	const fixedRecords = 4500

	response := HeavyResponse{
		Operation:       "heavy_inventory",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Preload("Inventory").Find(&products)

	var orders []models.Order
	h.db.Preload("Items").Preload("Payment").Find(&orders)

	var reviews []models.Review
	h.db.Find(&reviews)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 47185920 // 45 MB
	const fixedRecords = 15000

	response := HeavyResponse{
		Operation:       "heavy_aggregate",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var users []models.User
	h.db.Preload("Addresses").Preload("Orders").Preload("Orders.Items").Preload("Reviews").Find(&users)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 45088768 // 43 MB
	const fixedRecords = 3000

	response := HeavyResponse{
		Operation:       "heavy_users",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var categories []models.Category
	h.db.Preload("Products").Preload("Products.Reviews").Preload("Children").Find(&categories)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 37748736 // 36 MB
	const fixedRecords = 2000

	response := HeavyResponse{
		Operation:       "heavy_categories",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var payments []models.Payment
	h.db.Find(&payments)

	var orders []models.Order
	h.db.Preload("Items").Preload("User").Find(&orders)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 40894464 // 39 MB
	const fixedRecords = 7500

	response := HeavyResponse{
		Operation:       "heavy_payments",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var carts []models.Cart
	h.db.Preload("Items").Preload("Items.Product").Find(&carts)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 38797312 // 37 MB
	const fixedRecords = 6000

	response := HeavyResponse{
		Operation:       "heavy_carts",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
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

	// Fetch data from database (for actual DB interaction)
	var users []models.User
	h.db.Preload("Addresses").Preload("Orders").Preload("Reviews").Find(&users)

	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Preload("Inventory").Find(&products)

	var orders []models.Order
	h.db.Preload("Items").Preload("Payment").Preload("ShippingAddress").Find(&orders)

	var reviews []models.Review
	h.db.Find(&reviews)

	var categories []models.Category
	h.db.Find(&categories)

	var inventory []models.Inventory
	h.db.Find(&inventory)

	var payments []models.Payment
	h.db.Find(&payments)

	var carts []models.Cart
	h.db.Preload("Items").Find(&carts)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 52428800 // 50 MB
	const fixedRecords = 25000

	response := HeavyResponse{
		Operation:       "heavy_full_dump",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"users", "addresses", "products", "categories", "orders", "order_items", "payments", "reviews", "inventories", "carts", "cart_items"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy full dump operation completed - fetched all data from all tables",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyProductSearch simulates a heavy product search with full-text matching
func (h *HeavyHandler) HeavyProductSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var products []models.Product
	h.db.Preload("Category").Preload("Seller").Preload("Reviews").Where("name LIKE ? OR description LIKE ?", "%product%", "%quality%").Find(&products)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 46137344 // 44 MB
	const fixedRecords = 4800

	response := HeavyResponse{
		Operation:       "heavy_product_search",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"products", "categories", "users", "reviews"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy product search completed - full-text search across products",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyOrderHistory fetches complete order history with all related data
func (h *HeavyHandler) HeavyOrderHistory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var orders []models.Order
	h.db.Preload("Items").Preload("Items.Product").Preload("Payment").Preload("ShippingAddress").Preload("User").Order("created_at DESC").Find(&orders)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 48234496 // 46 MB
	const fixedRecords = 9500

	response := HeavyResponse{
		Operation:       "heavy_order_history",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"orders", "order_items", "products", "payments", "addresses", "users"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy order history completed - chronological order data with relations",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyUserActivity fetches all user activity including logins, actions, and engagement
func (h *HeavyHandler) HeavyUserActivity(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var users []models.User
	h.db.Preload("Addresses").Preload("Orders").Preload("Orders.Items").Preload("Reviews").Find(&users)

	var orders []models.Order
	h.db.Preload("Items").Find(&orders)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 49283072 // 47 MB
	const fixedRecords = 11000

	response := HeavyResponse{
		Operation:       "heavy_user_activity",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"users", "addresses", "orders", "order_items", "reviews", "carts"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy user activity completed - comprehensive user engagement data",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyAnalyticsDashboard fetches all data needed for analytics dashboard
func (h *HeavyHandler) HeavyAnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var products []models.Product
	h.db.Preload("Reviews").Preload("Inventory").Find(&products)

	var orders []models.Order
	h.db.Preload("Items").Preload("Payment").Find(&orders)

	var users []models.User
	h.db.Find(&users)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 51380224 // 49 MB
	const fixedRecords = 18000

	response := HeavyResponse{
		Operation:       "heavy_analytics_dashboard",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"products", "reviews", "inventories", "orders", "order_items", "payments", "users"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy analytics dashboard completed - comprehensive business metrics data",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavySalesTrends fetches historical sales data for trend analysis
func (h *HeavyHandler) HeavySalesTrends(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var orders []models.Order
	h.db.Preload("Items").Preload("Items.Product").Preload("Payment").Order("created_at").Find(&orders)

	var payments []models.Payment
	h.db.Find(&payments)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 43991040 // 42 MB
	const fixedRecords = 8200

	response := HeavyResponse{
		Operation:       "heavy_sales_trends",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"orders", "order_items", "products", "payments"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy sales trends completed - historical sales data for analysis",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyInventoryReport fetches complete inventory status across all warehouses
func (h *HeavyHandler) HeavyInventoryReport(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var inventory []models.Inventory
	h.db.Preload("Product").Preload("Product.Category").Find(&inventory)

	var products []models.Product
	h.db.Preload("Category").Find(&products)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 41943040 // 40 MB
	const fixedRecords = 5500

	response := HeavyResponse{
		Operation:       "heavy_inventory_report",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"inventories", "products", "categories"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy inventory report completed - full warehouse status data",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyReviewSentiment fetches all reviews for sentiment analysis
func (h *HeavyHandler) HeavyReviewSentiment(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var reviews []models.Review
	h.db.Preload("User").Preload("Product").Preload("Product.Category").Find(&reviews)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 44040192 // 42 MB
	const fixedRecords = 14000

	response := HeavyResponse{
		Operation:       "heavy_review_sentiment",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"reviews", "users", "products", "categories"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy review sentiment completed - review data for sentiment analysis",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyCategoryTree fetches complete category hierarchy with products
func (h *HeavyHandler) HeavyCategoryTree(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var categories []models.Category
	h.db.Preload("Products").Preload("Products.Reviews").Preload("Products.Inventory").Preload("Children").Find(&categories)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 39845888 // 38 MB
	const fixedRecords = 3500

	response := HeavyResponse{
		Operation:       "heavy_category_tree",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"categories", "products", "reviews", "inventories"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy category tree completed - full category hierarchy with products",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyShippingData fetches all shipping and address data
func (h *HeavyHandler) HeavyShippingData(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var orders []models.Order
	h.db.Preload("ShippingAddress").Preload("Items").Preload("User").Find(&orders)

	var users []models.User
	h.db.Preload("Addresses").Find(&users)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 45088768 // 43 MB
	const fixedRecords = 7000

	response := HeavyResponse{
		Operation:       "heavy_shipping_data",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"orders", "addresses", "order_items", "users"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy shipping data completed - comprehensive shipping information",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// for large size mocks
// HeavyFinancialSummary fetches all financial data for reporting
func (h *HeavyHandler) HeavyFinancialSummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Fetch data from database (for actual DB interaction)
	var payments []models.Payment
	h.db.Find(&payments)

	var orders []models.Order
	h.db.Preload("Items").Preload("Payment").Find(&orders)

	var products []models.Product
	h.db.Find(&products)

	elapsed := time.Since(start).Milliseconds()

	// Fixed deterministic values for consistent responses
	const fixedBytes int64 = 50331648 // 48 MB
	const fixedRecords = 16000

	response := HeavyResponse{
		Operation:       "heavy_financial_summary",
		DataPulledBytes: fixedBytes,
		DataPulledMB:    float64(fixedBytes) / 1024 / 1024,
		RecordCount:     fixedRecords,
		TablesAccessed:  []string{"payments", "orders", "order_items", "products"},
		QueryTimeMs:     elapsed,
		Message:         "Heavy financial summary completed - comprehensive financial data",
	}

	utils.JSONResponse(w, http.StatusOK, response)
}
