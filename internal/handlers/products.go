package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/middleware"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
	"gorm.io/gorm"
)

type ProductHandler struct {
	db *database.Database
}

func NewProductHandler(db *database.Database) *ProductHandler {
	return &ProductHandler{db: db}
}

// ListProducts returns paginated products with filters
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var products []models.Product
	var total int64

	query := h.db.Model(&models.Product{}).Where("status = ?", "active")

	// Full-text search
	if q := r.URL.Query().Get("q"); q != "" {
		searchTerm := "%" + q + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchTerm, searchTerm)
	}

	// Category filter (includes subcategories)
	if categoryID := r.URL.Query().Get("categoryId"); categoryID != "" {
		if catUUID, err := uuid.Parse(categoryID); err == nil {
			// Get all child category IDs
			var categoryIDs []uuid.UUID
			h.db.Raw("WITH RECURSIVE category_tree AS ("+
				"SELECT id FROM categories WHERE id = ? "+
				"UNION ALL "+
				"SELECT c.id FROM categories c INNER JOIN category_tree ct ON c.parent_id = ct.id"+
				") SELECT id FROM category_tree", catUUID).Scan(&categoryIDs)

			if len(categoryIDs) > 0 {
				query = query.Where("category_id IN ?", categoryIDs)
			} else {
				query = query.Where("category_id = ?", catUUID)
			}
		}
	}

	// Seller filter
	if sellerID := r.URL.Query().Get("sellerId"); sellerID != "" {
		if sellerUUID, err := uuid.Parse(sellerID); err == nil {
			query = query.Where("seller_id = ?", sellerUUID)
		}
	}

	// Price filters
	if minPrice := r.URL.Query().Get("minPrice"); minPrice != "" {
		query = query.Where("price >= ?", minPrice)
	}
	if maxPrice := r.URL.Query().Get("maxPrice"); maxPrice != "" {
		query = query.Where("price <= ?", maxPrice)
	}

	// In stock filter
	if inStock := r.URL.Query().Get("inStock"); inStock == "true" {
		query = query.Joins("LEFT JOIN inventories ON inventories.product_id = products.id").
			Where("inventories.quantity > inventories.reserved_qty")
	}

	// Sorting
	sortBy := r.URL.Query().Get("sortBy")
	switch sortBy {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	case "newest":
		query = query.Order("created_at DESC")
	default:
		query = query.Order("created_at DESC")
	}

	query.Count(&total)
	query.Preload("Category").Offset(offset).Limit(limit).Find(&products)

	// Enrich with review stats
	type productWithStats struct {
		models.Product
		AverageRating float64 `json:"averageRating"`
		ReviewCount   int     `json:"reviewCount"`
		InStock       bool    `json:"inStock"`
	}

	enrichedProducts := make([]productWithStats, len(products))
	for i, p := range products {
		var stats struct {
			AvgRating float64
			Count     int
		}
		h.db.Model(&models.Review{}).Where("product_id = ?", p.ID).
			Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as count").Scan(&stats)

		var stock int64
		h.db.Model(&models.Inventory{}).Where("product_id = ?", p.ID).
			Select("COALESCE(SUM(quantity - reserved_qty), 0)").Scan(&stock)

		enrichedProducts[i] = productWithStats{
			Product:       p,
			AverageRating: stats.AvgRating,
			ReviewCount:   stats.Count,
			InStock:       stock > 0,
		}
	}

	utils.PaginatedJSONResponse(w, enrichedProducts, page, limit, int(total))
}

// GetProduct returns a single product with relations
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid product ID format")
		return
	}

	var product models.Product
	query := h.db.Model(&models.Product{})

	// Handle includes
	includes := r.URL.Query().Get("include")
	if includes != "" {
		for _, include := range strings.Split(includes, ",") {
			switch strings.TrimSpace(include) {
			case "category":
				query = query.Preload("Category")
			case "seller":
				query = query.Preload("Seller")
			case "reviews":
				query = query.Preload("Reviews", func(db *gorm.DB) *gorm.DB {
					return db.Order("created_at DESC").Limit(10)
				})
			case "inventory":
				query = query.Preload("Inventory")
			}
		}
	} else {
		query = query.Preload("Category").Preload("Seller")
	}

	if err := query.First(&product, productID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Product not found")
		return
	}

	// Get review stats
	var reviewStats struct {
		AvgRating    float64
		TotalReviews int64
	}
	h.db.Model(&models.Review{}).Where("product_id = ?", productID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as total_reviews").Scan(&reviewStats)

	// Get total stock
	var totalStock int64
	h.db.Model(&models.Inventory{}).Where("product_id = ?", productID).
		Select("COALESCE(SUM(quantity - reserved_qty), 0)").Scan(&totalStock)

	response := map[string]interface{}{
		"product": product,
		"reviewStats": map[string]interface{}{
			"averageRating": reviewStats.AvgRating,
			"totalReviews":  reviewStats.TotalReviews,
		},
		"totalStock": totalStock,
	}

	utils.JSONResponse(w, http.StatusOK, response)
}

// CreateProduct creates a new product
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		Name           string                 `json:"name"`
		Description    string                 `json:"description"`
		Price          float64                `json:"price"`
		CompareAtPrice float64                `json:"compareAtPrice"`
		SKU            string                 `json:"sku"`
		CategoryID     string                 `json:"categoryId"`
		Images         []string               `json:"images"`
		Attributes     map[string]interface{} `json:"attributes"`
		Status         string                 `json:"status"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Validate
	if req.Name == "" || req.Price <= 0 || req.CategoryID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Name, price, and category are required")
		return
	}

	categoryID, err := uuid.Parse(req.CategoryID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid category ID")
		return
	}

	sellerID, _ := uuid.Parse(userCtx.UserID)

	// Convert images and attributes to JSON strings
	imagesJSON, _ := json.Marshal(req.Images)
	attrsJSON, _ := json.Marshal(req.Attributes)

	product := &models.Product{
		Name:           req.Name,
		Slug:           generateSlug(req.Name),
		Description:    req.Description,
		Price:          req.Price,
		CompareAtPrice: req.CompareAtPrice,
		SKU:            req.SKU,
		CategoryID:     categoryID,
		SellerID:       sellerID,
		Images:         string(imagesJSON),
		Attributes:     string(attrsJSON),
		Status:         "draft",
	}

	if req.Status == "active" {
		product.Status = "active"
	}

	if err := h.db.Create(product).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create product")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, product)
}

// UpdateProduct updates an existing product
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid product ID format")
		return
	}

	var updates map[string]interface{}
	if err := utils.ParseJSON(r, &updates); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Remove protected fields
	delete(updates, "id")
	delete(updates, "sellerId")
	delete(updates, "createdAt")

	// Handle JSON fields
	if images, ok := updates["images"].([]interface{}); ok {
		imagesJSON, _ := json.Marshal(images)
		updates["images"] = string(imagesJSON)
	}
	if attrs, ok := updates["attributes"].(map[string]interface{}); ok {
		attrsJSON, _ := json.Marshal(attrs)
		updates["attributes"] = string(attrsJSON)
	}

	if err := h.db.Model(&models.Product{}).Where("id = ?", productID).Updates(updates).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update product")
		return
	}

	var product models.Product
	h.db.First(&product, productID)
	utils.JSONResponse(w, http.StatusOK, product)
}

// DeleteProduct deletes a product
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid product ID format")
		return
	}

	if err := h.db.Delete(&models.Product{}, productID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetProductReviews returns reviews for a product
func (h *ProductHandler) GetProductReviews(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid product ID format")
		return
	}

	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var reviews []models.Review
	var total int64

	query := h.db.Model(&models.Review{}).Where("product_id = ?", productID)

	// Filter by rating
	if rating := r.URL.Query().Get("rating"); rating != "" {
		query = query.Where("rating = ?", rating)
	}

	// Filter by verified
	if verified := r.URL.Query().Get("verified"); verified == "true" {
		query = query.Where("verified_purchase = ?", true)
	}

	// Sorting
	sortBy := r.URL.Query().Get("sortBy")
	switch sortBy {
	case "oldest":
		query = query.Order("created_at ASC")
	case "highest_rating":
		query = query.Order("rating DESC")
	case "lowest_rating":
		query = query.Order("rating ASC")
	case "most_helpful":
		query = query.Order("helpful_count DESC")
	default:
		query = query.Order("created_at DESC")
	}

	query.Count(&total)
	query.Preload("User").Offset(offset).Limit(limit).Find(&reviews)

	// Get stats
	var stats struct {
		AvgRating       float64
		TotalReviews    int64
		VerifiedReviews int64
	}
	h.db.Model(&models.Review{}).Where("product_id = ?", productID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as total_reviews").Scan(&stats)
	h.db.Model(&models.Review{}).Where("product_id = ? AND verified_purchase = ?", productID, true).
		Count(&stats.VerifiedReviews)

	// Rating distribution
	var distribution []struct {
		Rating int
		Count  int64
	}
	h.db.Model(&models.Review{}).Where("product_id = ?", productID).
		Select("rating, COUNT(*) as count").Group("rating").Scan(&distribution)

	ratingDist := map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}
	for _, d := range distribution {
		ratingDist[string(rune('0'+d.Rating))] = d.Count
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"reviews": reviews,
		"stats": map[string]interface{}{
			"averageRating":      stats.AvgRating,
			"totalReviews":       stats.TotalReviews,
			"verifiedReviews":    stats.VerifiedReviews,
			"ratingDistribution": ratingDist,
		},
		"pagination": utils.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: int(total),
			TotalPages: (int(total) + limit - 1) / limit,
			HasNext:    page < (int(total)+limit-1)/limit,
			HasPrev:    page > 1,
		},
	})
}

// CreateProductReview creates a review for a product
func (h *ProductHandler) CreateProductReview(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "productId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid product ID format")
		return
	}

	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		Rating  int      `json:"rating"`
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Images  []string `json:"images"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Rating must be between 1 and 5")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	// Check if user has already reviewed this product
	var existingReview models.Review
	if err := h.db.Where("product_id = ? AND user_id = ?", productID, userID).First(&existingReview).Error; err == nil {
		utils.ErrorResponse(w, http.StatusConflict, "ALREADY_REVIEWED", "You have already reviewed this product")
		return
	}

	// Check if user has purchased this product
	var orderCount int64
	h.db.Model(&models.OrderItem{}).Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("order_items.product_id = ? AND orders.user_id = ? AND orders.status = ?", productID, userID, "delivered").
		Count(&orderCount)

	imagesJSON, _ := json.Marshal(req.Images)

	review := &models.Review{
		ProductID:        productID,
		UserID:           userID,
		Rating:           req.Rating,
		Title:            req.Title,
		Content:          req.Content,
		Images:           string(imagesJSON),
		VerifiedPurchase: orderCount > 0,
	}

	if err := h.db.Create(review).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create review")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, review)
}

// BulkCreateProducts creates multiple products
func (h *ProductHandler) BulkCreateProducts(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		Products []struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Price       float64  `json:"price"`
			SKU         string   `json:"sku"`
			CategoryID  string   `json:"categoryId"`
			Images      []string `json:"images"`
		} `json:"products"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	sellerID, _ := uuid.Parse(userCtx.UserID)
	successful := 0
	failed := 0
	var errors []map[string]interface{}

	for i, p := range req.Products {
		categoryID, err := uuid.Parse(p.CategoryID)
		if err != nil {
			failed++
			errors = append(errors, map[string]interface{}{"index": i, "error": "Invalid category ID"})
			continue
		}

		imagesJSON, _ := json.Marshal(p.Images)
		product := &models.Product{
			Name:        p.Name,
			Slug:        generateSlug(p.Name),
			Description: p.Description,
			Price:       p.Price,
			SKU:         p.SKU,
			CategoryID:  categoryID,
			SellerID:    sellerID,
			Images:      string(imagesJSON),
			Status:      "draft",
		}

		if err := h.db.Create(product).Error; err != nil {
			failed++
			errors = append(errors, map[string]interface{}{"index": i, "error": err.Error()})
		} else {
			successful++
		}
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"successful": successful,
		"failed":     failed,
		"errors":     errors,
	})
}

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Add UUID suffix to ensure uniqueness
	return slug + "-" + uuid.New().String()[:8]
}
