package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type CouponHandler struct {
	db *database.Database
}

func NewCouponHandler(db *database.Database) *CouponHandler {
	return &CouponHandler{db: db}
}

// ListCoupons returns paginated coupons
func (h *CouponHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var coupons []models.Coupon
	var total int64

	query := h.db.Model(&models.Coupon{})

	// Apply filters
	if active := r.URL.Query().Get("active"); active != "" {
		query = query.Where("is_active = ?", active == "true")
	}
	if couponType := r.URL.Query().Get("type"); couponType != "" {
		query = query.Where("type = ?", couponType)
	}

	query.Count(&total)
	query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&coupons)

	utils.PaginatedJSONResponse(w, coupons, page, limit, int(total))
}

// GetCoupon returns a coupon with usage stats
func (h *CouponHandler) GetCoupon(w http.ResponseWriter, r *http.Request) {
	couponID, err := uuid.Parse(chi.URLParam(r, "couponId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid coupon ID format")
		return
	}

	var coupon models.Coupon
	if err := h.db.First(&coupon, couponID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Coupon not found")
		return
	}

	// Get usage stats
	var totalDiscount float64
	h.db.Model(&models.Order{}).Where("coupon_id = ?", couponID).Select("COALESCE(SUM(discount), 0)").Scan(&totalDiscount)

	var uniqueUsers int64
	h.db.Model(&models.Order{}).Where("coupon_id = ?", couponID).Distinct("user_id").Count(&uniqueUsers)

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"coupon": coupon,
		"usageStats": map[string]interface{}{
			"totalUsage":         coupon.UsageCount,
			"totalDiscountGiven": totalDiscount,
			"uniqueUsers":        uniqueUsers,
		},
	})
}

// CreateCoupon creates a new coupon
func (h *CouponHandler) CreateCoupon(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code                 string   `json:"code"`
		Description          string   `json:"description"`
		Type                 string   `json:"type"`
		Value                float64  `json:"value"`
		MinOrderAmount       float64  `json:"minOrderAmount"`
		MaxDiscount          float64  `json:"maxDiscount"`
		UsageLimit           int      `json:"usageLimit"`
		PerUserLimit         int      `json:"perUserLimit"`
		ApplicableCategories []string `json:"applicableCategories"`
		ApplicableProducts   []string `json:"applicableProducts"`
		StartDate            string   `json:"startDate"`
		EndDate              string   `json:"endDate"`
		IsActive             bool     `json:"isActive"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Code == "" || req.Type == "" || req.Value <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Code, type, and value are required")
		return
	}

	validTypes := map[string]bool{"percentage": true, "fixed": true, "free_shipping": true}
	if !validTypes[req.Type] {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid coupon type")
		return
	}

	// Check if code exists
	var existing models.Coupon
	if err := h.db.Where("code = ?", req.Code).First(&existing).Error; err == nil {
		utils.ErrorResponse(w, http.StatusConflict, "CODE_EXISTS", "Coupon code already exists")
		return
	}

	coupon := &models.Coupon{
		Code:           req.Code,
		Description:    req.Description,
		Type:           req.Type,
		Value:          req.Value,
		MinOrderAmount: req.MinOrderAmount,
		MaxDiscount:    req.MaxDiscount,
		UsageLimit:     req.UsageLimit,
		PerUserLimit:   req.PerUserLimit,
		IsActive:       req.IsActive,
	}

	if req.StartDate != "" {
		if t, err := time.Parse(time.RFC3339, req.StartDate); err == nil {
			coupon.StartDate = &t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse(time.RFC3339, req.EndDate); err == nil {
			coupon.EndDate = &t
		}
	}

	if err := h.db.Create(coupon).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create coupon")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, coupon)
}

// UpdateCoupon updates an existing coupon
func (h *CouponHandler) UpdateCoupon(w http.ResponseWriter, r *http.Request) {
	couponID, err := uuid.Parse(chi.URLParam(r, "couponId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid coupon ID format")
		return
	}

	var updates map[string]interface{}
	if err := utils.ParseJSON(r, &updates); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	delete(updates, "id")
	delete(updates, "code") // Can't change code
	delete(updates, "createdAt")
	delete(updates, "usageCount") // Can't manually set usage count

	if err := h.db.Model(&models.Coupon{}).Where("id = ?", couponID).Updates(updates).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update coupon")
		return
	}

	var coupon models.Coupon
	h.db.First(&coupon, couponID)
	utils.JSONResponse(w, http.StatusOK, coupon)
}

// DeleteCoupon deletes a coupon
func (h *CouponHandler) DeleteCoupon(w http.ResponseWriter, r *http.Request) {
	couponID, err := uuid.Parse(chi.URLParam(r, "couponId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid coupon ID format")
		return
	}

	if err := h.db.Delete(&models.Coupon{}, couponID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete coupon")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ValidateCoupon validates a coupon code
func (h *CouponHandler) ValidateCoupon(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code       string   `json:"code"`
		CartTotal  float64  `json:"cartTotal"`
		ProductIDs []string `json:"productIds"`
	}

	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Code == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Coupon code is required")
		return
	}

	var coupon models.Coupon
	if err := h.db.Where("code = ?", req.Code).First(&coupon).Error; err != nil {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Invalid coupon code",
		})
		return
	}

	// Validate coupon
	now := time.Now()

	if !coupon.IsActive {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Coupon is not active",
		})
		return
	}

	if coupon.StartDate != nil && now.Before(*coupon.StartDate) {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Coupon is not yet active",
		})
		return
	}

	if coupon.EndDate != nil && now.After(*coupon.EndDate) {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Coupon has expired",
		})
		return
	}

	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Coupon usage limit reached",
		})
		return
	}

	if coupon.MinOrderAmount > 0 && req.CartTotal < coupon.MinOrderAmount {
		utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
			"valid":   false,
			"message": "Order does not meet minimum amount",
		})
		return
	}

	// Calculate discount
	var discount float64
	switch coupon.Type {
	case "percentage":
		discount = req.CartTotal * (coupon.Value / 100)
	case "fixed":
		discount = coupon.Value
	case "free_shipping":
		discount = 9.99 // Standard shipping cost
	}

	if coupon.MaxDiscount > 0 && discount > coupon.MaxDiscount {
		discount = coupon.MaxDiscount
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"valid":    true,
		"coupon":   coupon,
		"discount": discount,
		"message":  "Coupon is valid",
	})
}
