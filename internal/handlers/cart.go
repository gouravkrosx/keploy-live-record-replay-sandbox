package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/middleware"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type CartHandler struct {
	db *database.Database
}

func NewCartHandler(db *database.Database) *CartHandler {
	return &CartHandler{db: db}
}

type CartResponse struct {
	ID                uuid.UUID          `json:"id"`
	Items             []CartItemResponse `json:"items"`
	Subtotal          float64            `json:"subtotal"`
	Discount          float64            `json:"discount"`
	EstimatedShipping float64            `json:"estimatedShipping"`
	EstimatedTax      float64            `json:"estimatedTax"`
	Total             float64            `json:"total"`
	ItemCount         int                `json:"itemCount"`
	Coupon            *models.Coupon     `json:"coupon,omitempty"`
}

type CartItemResponse struct {
	ID                uuid.UUID      `json:"id"`
	ProductID         uuid.UUID      `json:"productId"`
	Quantity          int            `json:"quantity"`
	Product           models.Product `json:"product"`
	InStock           bool           `json:"inStock"`
	AvailableQuantity int            `json:"availableQuantity"`
}

// GetCart returns the current user's cart
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	cart := h.getOrCreateCart(userID)

	response := h.buildCartResponse(cart)
	utils.JSONResponse(w, http.StatusOK, response)
}

// ClearCart removes all items from cart
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	var cart models.Cart
	if err := h.db.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.db.Where("cart_id = ?", cart.ID).Delete(&models.CartItem{})
	h.db.Model(&cart).Update("coupon_id", nil)

	w.WriteHeader(http.StatusNoContent)
}

// AddCartItem adds an item to the cart
func (h *CartHandler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.ProductID == "" || req.Quantity < 1 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Product ID and quantity are required")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid product ID")
		return
	}

	// Check if product exists and is active
	var product models.Product
	if err := h.db.Where("id = ? AND status = ?", productID, "active").First(&product).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Product not found or not available")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	cart := h.getOrCreateCart(userID)

	// Check if item already in cart
	var existingItem models.CartItem
	if err := h.db.Where("cart_id = ? AND product_id = ?", cart.ID, productID).First(&existingItem).Error; err == nil {
		// Update quantity
		existingItem.Quantity += req.Quantity
		h.db.Save(&existingItem)
	} else {
		// Add new item
		cartItem := &models.CartItem{
			CartID:    cart.ID,
			ProductID: productID,
			Quantity:  req.Quantity,
		}
		h.db.Create(cartItem)
	}

	// Reload cart
	h.db.Preload("Items.Product").Preload("Coupon").First(cart, cart.ID)
	response := h.buildCartResponse(cart)
	utils.JSONResponse(w, http.StatusOK, response)
}

// UpdateCartItem updates the quantity of a cart item
func (h *CartHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid item ID format")
		return
	}

	var req struct {
		Quantity int `json:"quantity"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Quantity < 1 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Quantity must be at least 1")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	// Verify item belongs to user's cart
	var cart models.Cart
	if err := h.db.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Cart not found")
		return
	}

	var cartItem models.CartItem
	if err := h.db.Where("id = ? AND cart_id = ?", itemID, cart.ID).First(&cartItem).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Cart item not found")
		return
	}

	cartItem.Quantity = req.Quantity
	h.db.Save(&cartItem)

	// Reload cart
	h.db.Preload("Items.Product").Preload("Coupon").First(&cart, cart.ID)
	response := h.buildCartResponse(&cart)
	utils.JSONResponse(w, http.StatusOK, response)
}

// RemoveCartItem removes an item from the cart
func (h *CartHandler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	itemID, err := uuid.Parse(chi.URLParam(r, "itemId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid item ID format")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	var cart models.Cart
	if err := h.db.Where("user_id = ?", userID).First(&cart).Error; err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	h.db.Where("id = ? AND cart_id = ?", itemID, cart.ID).Delete(&models.CartItem{})

	w.WriteHeader(http.StatusNoContent)
}

// ValidateCart validates the cart
func (h *CartHandler) ValidateCart(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)

	var cart models.Cart
	if err := h.db.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{"valid": true, "issues": []interface{}{}})
		return
	}

	var issues []map[string]interface{}

	for _, item := range cart.Items {
		// Check if product is still active
		if item.Product.Status != "active" {
			issues = append(issues, map[string]interface{}{
				"type":      "product_unavailable",
				"productId": item.ProductID.String(),
				"message":   "Product is no longer available",
			})
			continue
		}

		// Check stock
		var totalStock int64
		h.db.Model(&models.Inventory{}).Where("product_id = ?", item.ProductID).
			Select("COALESCE(SUM(quantity - reserved_qty), 0)").Scan(&totalStock)

		if totalStock == 0 {
			issues = append(issues, map[string]interface{}{
				"type":      "out_of_stock",
				"productId": item.ProductID.String(),
				"message":   "Product is out of stock",
			})
		} else if int(totalStock) < item.Quantity {
			issues = append(issues, map[string]interface{}{
				"type":      "insufficient_stock",
				"productId": item.ProductID.String(),
				"message":   "Only " + string(rune(totalStock)) + " items available",
				"details":   map[string]int64{"available": totalStock},
			})
		}
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"valid":  len(issues) == 0,
		"issues": issues,
	})
}

// ApplyCoupon applies a coupon to the cart
func (h *CartHandler) ApplyCoupon(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		Code string `json:"code"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Code == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Coupon code is required")
		return
	}

	// Find coupon
	var coupon models.Coupon
	if err := h.db.Where("code = ? AND is_active = ?", req.Code, true).First(&coupon).Error; err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_COUPON", "Invalid coupon code")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	cart := h.getOrCreateCart(userID)

	// Apply coupon
	h.db.Model(&cart).Update("coupon_id", coupon.ID)

	// Reload cart
	h.db.Preload("Items.Product").Preload("Coupon").First(cart, cart.ID)
	response := h.buildCartResponse(cart)
	utils.JSONResponse(w, http.StatusOK, response)
}

func (h *CartHandler) getOrCreateCart(userID uuid.UUID) *models.Cart {
	var cart models.Cart
	if err := h.db.Preload("Items.Product").Preload("Coupon").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		cart = models.Cart{UserID: userID}
		h.db.Create(&cart)
	}
	return &cart
}

func (h *CartHandler) buildCartResponse(cart *models.Cart) *CartResponse {
	var subtotal float64
	items := make([]CartItemResponse, len(cart.Items))

	for i, item := range cart.Items {
		subtotal += item.Product.Price * float64(item.Quantity)

		// Get available stock
		var stock int64
		h.db.Model(&models.Inventory{}).Where("product_id = ?", item.ProductID).
			Select("COALESCE(SUM(quantity - reserved_qty), 0)").Scan(&stock)

		items[i] = CartItemResponse{
			ID:                item.ID,
			ProductID:         item.ProductID,
			Quantity:          item.Quantity,
			Product:           item.Product,
			InStock:           stock > 0,
			AvailableQuantity: int(stock),
		}
	}

	var discount float64
	if cart.Coupon != nil {
		switch cart.Coupon.Type {
		case "percentage":
			discount = subtotal * (cart.Coupon.Value / 100)
		case "fixed":
			discount = cart.Coupon.Value
		}
		if cart.Coupon.MaxDiscount > 0 && discount > cart.Coupon.MaxDiscount {
			discount = cart.Coupon.MaxDiscount
		}
	}

	shipping := 9.99
	if subtotal > 100 {
		shipping = 0
	}
	tax := (subtotal - discount) * 0.08

	return &CartResponse{
		ID:                cart.ID,
		Items:             items,
		Subtotal:          subtotal,
		Discount:          discount,
		EstimatedShipping: shipping,
		EstimatedTax:      tax,
		Total:             subtotal - discount + shipping + tax,
		ItemCount:         len(cart.Items),
		Coupon:            cart.Coupon,
	}
}
