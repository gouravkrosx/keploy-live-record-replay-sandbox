package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/middleware"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type OrderHandler struct {
	db *database.Database
}

func NewOrderHandler(db *database.Database) *OrderHandler {
	return &OrderHandler{db: db}
}

// ListOrders returns paginated orders (admin)
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var orders []models.Order
	var total int64

	query := h.db.Model(&models.Order{})

	// Apply filters
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if minTotal := r.URL.Query().Get("minTotal"); minTotal != "" {
		query = query.Where("total >= ?", minTotal)
	}
	if maxTotal := r.URL.Query().Get("maxTotal"); maxTotal != "" {
		query = query.Where("total <= ?", maxTotal)
	}

	query.Count(&total)
	query.Preload("Items").Preload("User").Preload("Payment").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&orders)

	utils.PaginatedJSONResponse(w, orders, page, limit, int(total))
}

// GetOrder returns a single order with all details
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid order ID format")
		return
	}

	var order models.Order
	if err := h.db.Preload("Items.Product").Preload("User").Preload("Payment").
		Preload("ShippingAddress").Preload("Coupon").First(&order, orderID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Order not found")
		return
	}

	// Check authorization
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx != nil && userCtx.Role != "admin" && userCtx.UserID != order.UserID.String() {
		utils.ErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Cannot view this order")
		return
	}

	utils.JSONResponse(w, http.StatusOK, order)
}

// CreateOrder creates a new order from cart
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req struct {
		ShippingAddressID string `json:"shippingAddressId"`
		BillingAddressID  string `json:"billingAddressId"`
		PaymentMethod     string `json:"paymentMethod"`
		Notes             string `json:"notes"`
		CouponCode        string `json:"couponCode"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.ShippingAddressID == "" || req.PaymentMethod == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Shipping address and payment method are required")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	shippingAddressID, _ := uuid.Parse(req.ShippingAddressID)

	// Get user's cart
	var cart models.Cart
	if err := h.db.Preload("Items.Product").Where("user_id = ?", userID).First(&cart).Error; err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "EMPTY_CART", "Cart is empty")
		return
	}

	if len(cart.Items) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "EMPTY_CART", "Cart is empty")
		return
	}

	// Calculate totals
	var subtotal float64
	for _, item := range cart.Items {
		subtotal += item.Product.Price * float64(item.Quantity)
	}

	var discount float64
	var couponID *uuid.UUID

	// Apply coupon if provided
	if req.CouponCode != "" {
		var coupon models.Coupon
		if err := h.db.Where("code = ? AND is_active = ?", req.CouponCode, true).First(&coupon).Error; err == nil {
			// Validate coupon
			now := time.Now()
			if coupon.StartDate != nil && now.Before(*coupon.StartDate) {
				utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_COUPON", "Coupon is not yet active")
				return
			}
			if coupon.EndDate != nil && now.After(*coupon.EndDate) {
				utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_COUPON", "Coupon has expired")
				return
			}
			if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
				utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_COUPON", "Coupon usage limit reached")
				return
			}
			if coupon.MinOrderAmount > 0 && subtotal < coupon.MinOrderAmount {
				utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_COUPON", "Order does not meet minimum amount")
				return
			}

			// Calculate discount
			switch coupon.Type {
			case "percentage":
				discount = subtotal * (coupon.Value / 100)
			case "fixed":
				discount = coupon.Value
			case "free_shipping":
				// Will be handled below
			}

			if coupon.MaxDiscount > 0 && discount > coupon.MaxDiscount {
				discount = coupon.MaxDiscount
			}

			couponID = &coupon.ID
		}
	}

	// Calculate shipping and tax
	shippingCost := 9.99
	if subtotal > 100 {
		shippingCost = 0 // Free shipping for orders over $100
	}
	tax := (subtotal - discount) * 0.08 // 8% tax

	total := subtotal - discount + shippingCost + tax

	// Generate order number
	orderNumber := fmt.Sprintf("ORD-%d-%s", time.Now().Unix(), uuid.New().String()[:8])

	// Create order
	order := &models.Order{
		OrderNumber:       orderNumber,
		UserID:            userID,
		Status:            "pending",
		Subtotal:          subtotal,
		Discount:          discount,
		ShippingCost:      shippingCost,
		Tax:               tax,
		Total:             total,
		CouponID:          couponID,
		ShippingAddressID: shippingAddressID,
		Notes:             req.Notes,
	}

	if req.BillingAddressID != "" {
		billingID, _ := uuid.Parse(req.BillingAddressID)
		order.BillingAddressID = &billingID
	}

	tx := h.db.Begin()

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create order")
		return
	}

	// Create order items
	for _, cartItem := range cart.Items {
		orderItem := &models.OrderItem{
			OrderID:      order.ID,
			ProductID:    cartItem.ProductID,
			ProductName:  cartItem.Product.Name,
			ProductImage: cartItem.Product.Images,
			SKU:          cartItem.Product.SKU,
			Quantity:     cartItem.Quantity,
			UnitPrice:    cartItem.Product.Price,
			TotalPrice:   cartItem.Product.Price * float64(cartItem.Quantity),
		}
		if err := tx.Create(orderItem).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create order items")
			return
		}

		// Reserve inventory
		tx.Model(&models.Inventory{}).Where("product_id = ?", cartItem.ProductID).
			Update("reserved_qty", func() interface{} { return cartItem.Quantity }())
	}

	// Create payment record
	payment := &models.Payment{
		OrderID:  order.ID,
		Amount:   total,
		Currency: "USD",
		Method:   req.PaymentMethod,
		Status:   "pending",
	}
	if err := tx.Create(payment).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create payment")
		return
	}

	// Update coupon usage
	if couponID != nil {
		tx.Model(&models.Coupon{}).Where("id = ?", *couponID).Update("usage_count", func() interface{} { return 1 }())
	}

	// Clear cart
	tx.Where("cart_id = ?", cart.ID).Delete(&models.CartItem{})

	tx.Commit()

	// Reload order with relations
	h.db.Preload("Items").Preload("Payment").Preload("ShippingAddress").First(order, order.ID)

	utils.JSONResponse(w, http.StatusCreated, order)
}

// UpdateOrderStatus updates order status
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid order ID format")
		return
	}

	var req struct {
		Status         string `json:"status"`
		Note           string `json:"note"`
		TrackingNumber string `json:"trackingNumber"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	validStatuses := map[string]bool{
		"confirmed": true, "processing": true, "shipped": true, "delivered": true, "cancelled": true,
	}
	if !validStatuses[req.Status] {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid status")
		return
	}

	updates := map[string]interface{}{"status": req.Status}

	if err := h.db.Model(&models.Order{}).Where("id = ?", orderID).Updates(updates).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update order")
		return
	}

	// If delivered, update payment status
	if req.Status == "delivered" {
		h.db.Model(&models.Payment{}).Where("order_id = ?", orderID).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": time.Now(),
		})
	}

	var order models.Order
	h.db.Preload("Items").Preload("Payment").First(&order, orderID)
	utils.JSONResponse(w, http.StatusOK, order)
}

// CancelOrder cancels an order
func (h *OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid order ID format")
		return
	}

	var order models.Order
	if err := h.db.First(&order, orderID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Order not found")
		return
	}

	// Check if order can be cancelled
	if order.Status == "shipped" || order.Status == "delivered" {
		utils.ErrorResponse(w, http.StatusBadRequest, "CANNOT_CANCEL", "Cannot cancel shipped or delivered orders")
		return
	}

	// Update order status
	h.db.Model(&order).Update("status", "cancelled")

	// Release inventory reservations
	var orderItems []models.OrderItem
	h.db.Where("order_id = ?", orderID).Find(&orderItems)
	for _, item := range orderItems {
		h.db.Model(&models.Inventory{}).Where("product_id = ?", item.ProductID).
			Update("reserved_qty", func() interface{} { return -item.Quantity }())
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": "Order cancelled successfully"})
}

// RefundOrder processes a refund
func (h *OrderHandler) RefundOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid order ID format")
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	var order models.Order
	if err := h.db.Preload("Payment").First(&order, orderID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Order not found")
		return
	}

	refundAmount := req.Amount
	if refundAmount <= 0 {
		refundAmount = order.Total
	}

	if refundAmount > order.Total-order.Payment.RefundedAmount {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_AMOUNT", "Refund amount exceeds available balance")
		return
	}

	// Update payment
	newRefundedAmount := order.Payment.RefundedAmount + refundAmount
	status := "partially_refunded"
	if newRefundedAmount >= order.Total {
		status = "refunded"
	}

	h.db.Model(&order.Payment).Updates(map[string]interface{}{
		"refunded_amount": newRefundedAmount,
		"status":          status,
	})

	if newRefundedAmount >= order.Total {
		h.db.Model(&order).Update("status", "refunded")
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":        "Refund processed successfully",
		"refundedAmount": refundAmount,
		"totalRefunded":  newRefundedAmount,
	})
}
