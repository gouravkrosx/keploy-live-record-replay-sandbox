package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/middleware"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type UserHandler struct {
	db *database.Database
}

func NewUserHandler(db *database.Database) *UserHandler {
	return &UserHandler{db: db}
}

// ListUsers returns paginated list of users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var users []models.User
	var total int64

	query := h.db.Model(&models.User{})

	// Apply filters
	if role := r.URL.Query().Get("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Offset(offset).Limit(limit).Find(&users)

	utils.PaginatedJSONResponse(w, users, page, limit, int(total))
}

// GetUser returns a single user by ID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID format")
		return
	}

	var user models.User
	query := h.db.Model(&models.User{})

	// Handle includes
	includes := r.URL.Query().Get("include")
	if includes != "" {
		for _, include := range strings.Split(includes, ",") {
			switch strings.TrimSpace(include) {
			case "addresses":
				query = query.Preload("Addresses")
			case "orders":
				query = query.Preload("Orders")
			case "reviews":
				query = query.Preload("Reviews")
			case "cart":
				query = query.Preload("Cart.Items.Product")
			}
		}
	}

	if err := query.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	utils.JSONResponse(w, http.StatusOK, user)
}

// CreateUser creates a new user (admin only)
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := utils.ParseJSON(r, &user); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if err := h.db.Create(&user).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, user)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID format")
		return
	}

	var updates map[string]interface{}
	if err := utils.ParseJSON(r, &updates); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Remove sensitive fields
	delete(updates, "id")
	delete(updates, "password")
	delete(updates, "email")

	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update user")
		return
	}

	var user models.User
	h.db.First(&user, userID)
	utils.JSONResponse(w, http.StatusOK, user)
}

// DeleteUser deletes a user (soft delete)
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID format")
		return
	}

	if err := h.db.Delete(&models.User{}, userID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUserOrders returns orders for a specific user
func (h *UserHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID format")
		return
	}

	// Check authorization - users can only see their own orders
	userCtx := middleware.GetUserFromContext(r.Context())
	if userCtx != nil && userCtx.Role != "admin" && userCtx.UserID != userID.String() {
		utils.ErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Cannot view other user's orders")
		return
	}

	page, limit := utils.GetPaginationParams(r)
	offset := (page - 1) * limit

	var orders []models.Order
	var total int64

	query := h.db.Model(&models.Order{}).Where("user_id = ?", userID)

	// Apply filters
	if status := r.URL.Query().Get("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Preload("Items").Offset(offset).Limit(limit).Order("created_at DESC").Find(&orders)

	// Calculate stats
	var stats struct {
		TotalOrders int64
		TotalSpent  float64
	}
	h.db.Model(&models.Order{}).Where("user_id = ?", userID).Count(&stats.TotalOrders)
	h.db.Model(&models.Order{}).Where("user_id = ?", userID).Select("COALESCE(SUM(total), 0)").Scan(&stats.TotalSpent)

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"stats": map[string]interface{}{
			"totalOrders": stats.TotalOrders,
			"totalSpent":  stats.TotalSpent,
			"averageOrderValue": func() float64 {
				if stats.TotalOrders > 0 {
					return stats.TotalSpent / float64(stats.TotalOrders)
				}
				return 0
			}(),
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

// ListUserAddresses returns addresses for a user
func (h *UserHandler) ListUserAddresses(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID format")
		return
	}

	var addresses []models.Address
	h.db.Where("user_id = ?", userID).Find(&addresses)

	utils.JSONResponse(w, http.StatusOK, addresses)
}

// CreateUserAddress creates a new address for a user
func (h *UserHandler) CreateUserAddress(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid user ID format")
		return
	}

	var address models.Address
	if err := utils.ParseJSON(r, &address); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	address.UserID = userID

	// If this is the first address or marked as default, set it as default
	if address.IsDefault {
		h.db.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false)
	}

	if err := h.db.Create(&address).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create address")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, address)
}
