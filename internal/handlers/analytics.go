package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type AnalyticsHandler struct {
	db *database.Database
}

func NewAnalyticsHandler(db *database.Database) *AnalyticsHandler {
	return &AnalyticsHandler{db: db}
}

// GetSalesAnalytics returns sales analytics
func (h *AnalyticsHandler) GetSalesAnalytics(w http.ResponseWriter, r *http.Request) {
	fromDate := r.URL.Query().Get("fromDate")
	toDate := r.URL.Query().Get("toDate")

	if fromDate == "" || toDate == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "fromDate and toDate are required")
		return
	}

	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)
	to = to.Add(24 * time.Hour) // Include end date

	// Summary stats
	var summary struct {
		TotalRevenue float64
		TotalOrders  int64
		TotalItems   int64
	}

	h.db.Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ? AND status NOT IN ?", from, to, []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(total), 0) as total_revenue, COUNT(*) as total_orders").
		Scan(&summary)

	h.db.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.created_at >= ? AND orders.created_at < ? AND orders.status NOT IN ?", from, to, []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&summary.TotalItems)

	avgOrderValue := float64(0)
	if summary.TotalOrders > 0 {
		avgOrderValue = summary.TotalRevenue / float64(summary.TotalOrders)
	}

	// Daily breakdown
	groupBy := r.URL.Query().Get("groupBy")
	if groupBy == "" {
		groupBy = "day"
	}

	var breakdown []struct {
		Period  string  `json:"period"`
		Revenue float64 `json:"revenue"`
		Orders  int64   `json:"orders"`
	}

	dateFormat := "%Y-%m-%d"
	if groupBy == "week" {
		dateFormat = "%Y-%u"
	} else if groupBy == "month" {
		dateFormat = "%Y-%m"
	}

	h.db.Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ? AND status NOT IN ?", from, to, []string{"cancelled", "refunded"}).
		Select("DATE_FORMAT(created_at, ?) as period, COALESCE(SUM(total), 0) as revenue, COUNT(*) as orders", dateFormat).
		Group("period").
		Order("period").
		Scan(&breakdown)

	// Previous period comparison
	periodDuration := to.Sub(from)
	prevFrom := from.Add(-periodDuration)
	prevTo := from

	var prevStats struct {
		TotalRevenue float64
		TotalOrders  int64
	}
	h.db.Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ? AND status NOT IN ?", prevFrom, prevTo, []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(total), 0) as total_revenue, COUNT(*) as total_orders").
		Scan(&prevStats)

	revenueChange := float64(0)
	ordersChange := float64(0)
	if prevStats.TotalRevenue > 0 {
		revenueChange = ((summary.TotalRevenue - prevStats.TotalRevenue) / prevStats.TotalRevenue) * 100
	}
	if prevStats.TotalOrders > 0 {
		ordersChange = ((float64(summary.TotalOrders) - float64(prevStats.TotalOrders)) / float64(prevStats.TotalOrders)) * 100
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"summary": map[string]interface{}{
			"totalRevenue":      summary.TotalRevenue,
			"totalOrders":       summary.TotalOrders,
			"averageOrderValue": avgOrderValue,
			"totalItemsSold":    summary.TotalItems,
		},
		"breakdown": breakdown,
		"comparison": map[string]interface{}{
			"revenueChange": revenueChange,
			"ordersChange":  ordersChange,
		},
	})
}

// GetBestsellers returns bestselling products
func (h *AnalyticsHandler) GetBestsellers(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}

	var from time.Time
	switch period {
	case "day":
		from = time.Now().AddDate(0, 0, -1)
	case "week":
		from = time.Now().AddDate(0, 0, -7)
	case "year":
		from = time.Now().AddDate(-1, 0, 0)
	default:
		from = time.Now().AddDate(0, -1, 0)
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		// Parse limit
	}

	var bestsellers []struct {
		ProductID uuid.UUID `json:"productId"`
		UnitsSold int64     `json:"unitsSold"`
		Revenue   float64   `json:"revenue"`
	}

	query := h.db.Model(&models.OrderItem{}).
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.created_at >= ? AND orders.status NOT IN ?", from, []string{"cancelled", "refunded"}).
		Select("order_items.product_id, SUM(order_items.quantity) as units_sold, SUM(order_items.total_price) as revenue").
		Group("order_items.product_id").
		Order("units_sold DESC").
		Limit(limit)

	// Filter by category
	if categoryID := r.URL.Query().Get("categoryId"); categoryID != "" {
		if catUUID, err := uuid.Parse(categoryID); err == nil {
			query = query.Joins("JOIN products ON products.id = order_items.product_id").
				Where("products.category_id = ?", catUUID)
		}
	}

	query.Scan(&bestsellers)

	// Enrich with product details
	type bestsellerWithProduct struct {
		Product   models.Product `json:"product"`
		UnitsSold int64          `json:"unitsSold"`
		Revenue   float64        `json:"revenue"`
		Rank      int            `json:"rank"`
	}

	result := make([]bestsellerWithProduct, len(bestsellers))
	for i, bs := range bestsellers {
		var product models.Product
		h.db.First(&product, bs.ProductID)
		result[i] = bestsellerWithProduct{
			Product:   product,
			UnitsSold: bs.UnitsSold,
			Revenue:   bs.Revenue,
			Rank:      i + 1,
		}
	}

	utils.JSONResponse(w, http.StatusOK, result)
}

// GetCustomerAnalytics returns customer analytics
func (h *AnalyticsHandler) GetCustomerAnalytics(w http.ResponseWriter, r *http.Request) {
	fromDate := r.URL.Query().Get("fromDate")
	toDate := r.URL.Query().Get("toDate")

	if fromDate == "" || toDate == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "fromDate and toDate are required")
		return
	}

	from, _ := time.Parse("2006-01-02", fromDate)
	to, _ := time.Parse("2006-01-02", toDate)
	to = to.Add(24 * time.Hour)

	// Total customers
	var totalCustomers int64
	h.db.Model(&models.User{}).Where("role = ?", "customer").Count(&totalCustomers)

	// New customers in period
	var newCustomers int64
	h.db.Model(&models.User{}).Where("role = ? AND created_at >= ? AND created_at < ?", "customer", from, to).Count(&newCustomers)

	// Customers who made orders
	var activeCustomers int64
	h.db.Model(&models.Order{}).
		Where("created_at >= ? AND created_at < ?", from, to).
		Distinct("user_id").Count(&activeCustomers)

	// Returning customers (more than 1 order)
	var returningCustomers int64
	h.db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT user_id FROM orders 
			WHERE created_at >= ? AND created_at < ?
			GROUP BY user_id 
			HAVING COUNT(*) > 1
		) as returning_users
	`, from, to).Scan(&returningCustomers)

	// Average lifetime value
	var avgLTV float64
	h.db.Model(&models.Order{}).
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(AVG(user_totals.total_spent), 0)").
		Joins(`JOIN (SELECT user_id, SUM(total) as total_spent FROM orders GROUP BY user_id) as user_totals ON 1=1`).
		Limit(1).
		Scan(&avgLTV)

	h.db.Raw(`
		SELECT COALESCE(AVG(total_spent), 0) FROM (
			SELECT user_id, SUM(total) as total_spent 
			FROM orders 
			WHERE status NOT IN ('cancelled', 'refunded')
			GROUP BY user_id
		) as user_totals
	`).Scan(&avgLTV)

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"summary": map[string]interface{}{
			"totalCustomers":       totalCustomers,
			"newCustomers":         newCustomers,
			"returningCustomers":   returningCustomers,
			"averageLifetimeValue": avgLTV,
		},
		"segmentation": map[string]interface{}{
			"byOrderCount": []map[string]interface{}{
				{"segment": "1 order", "description": "New buyers"},
				{"segment": "2-5 orders", "description": "Regular buyers"},
				{"segment": "6+ orders", "description": "Loyal customers"},
			},
		},
	})
}

// GetInventoryAlerts returns inventory alerts
func (h *AnalyticsHandler) GetInventoryAlerts(w http.ResponseWriter, r *http.Request) {
	// Low stock items
	var lowStock []struct {
		ProductID       uuid.UUID `json:"productId"`
		WarehouseID     uuid.UUID `json:"warehouseId"`
		WarehouseName   string    `json:"warehouseName"`
		CurrentQuantity int       `json:"currentQuantity"`
		ReorderLevel    int       `json:"reorderLevel"`
	}

	h.db.Model(&models.Inventory{}).
		Where("quantity - reserved_qty <= reorder_level AND quantity - reserved_qty > 0").
		Select("product_id, warehouse_id, warehouse_name, quantity - reserved_qty as current_quantity, reorder_level").
		Scan(&lowStock)

	// Enrich with product info
	type alertWithProduct struct {
		Product         models.Product `json:"product"`
		WarehouseID     uuid.UUID      `json:"warehouseId"`
		WarehouseName   string         `json:"warehouseName"`
		CurrentQuantity int            `json:"currentQuantity"`
		ReorderLevel    int            `json:"reorderLevel"`
	}

	lowStockAlerts := make([]alertWithProduct, len(lowStock))
	for i, ls := range lowStock {
		var product models.Product
		h.db.First(&product, ls.ProductID)
		lowStockAlerts[i] = alertWithProduct{
			Product:         product,
			WarehouseID:     ls.WarehouseID,
			WarehouseName:   ls.WarehouseName,
			CurrentQuantity: ls.CurrentQuantity,
			ReorderLevel:    ls.ReorderLevel,
		}
	}

	// Out of stock
	var outOfStock []struct {
		ProductID   uuid.UUID  `json:"productId"`
		LastInStock *time.Time `json:"lastInStock"`
	}

	h.db.Model(&models.Inventory{}).
		Where("quantity - reserved_qty <= 0").
		Select("product_id, MAX(updated_at) as last_in_stock").
		Group("product_id").
		Scan(&outOfStock)

	type outOfStockAlert struct {
		Product     models.Product `json:"product"`
		LastInStock *time.Time     `json:"lastInStock"`
	}

	outOfStockAlerts := make([]outOfStockAlert, len(outOfStock))
	for i, oos := range outOfStock {
		var product models.Product
		h.db.First(&product, oos.ProductID)
		outOfStockAlerts[i] = outOfStockAlert{
			Product:     product,
			LastInStock: oos.LastInStock,
		}
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"lowStock":   lowStockAlerts,
		"outOfStock": outOfStockAlerts,
	})
}
