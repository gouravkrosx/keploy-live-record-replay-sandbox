package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/marketplace-api/internal/auth"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/middleware"
	"github.com/marketplace-api/internal/models"
	"github.com/marketplace-api/internal/utils"
)

type AuthHandler struct {
	db         *database.Database
	jwtService *auth.JWTService
}

func NewAuthHandler(db *database.Database, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{db: db, jwtService: jwtService}
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone,omitempty"`
	Role      string `json:"role,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type PasswordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	TokenType    string       `json:"tokenType"`
	ExpiresIn    int          `json:"expiresIn"`
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Validate required fields
	var validationErrors []utils.ErrorDetail
	if req.Email == "" {
		validationErrors = append(validationErrors, utils.ErrorDetail{Field: "email", Message: "Email is required"})
	}
	if req.Password == "" || len(req.Password) < 8 {
		validationErrors = append(validationErrors, utils.ErrorDetail{Field: "password", Message: "Password must be at least 8 characters"})
	}
	if req.FirstName == "" {
		validationErrors = append(validationErrors, utils.ErrorDetail{Field: "firstName", Message: "First name is required"})
	}
	if req.LastName == "" {
		validationErrors = append(validationErrors, utils.ErrorDetail{Field: "lastName", Message: "Last name is required"})
	}
	if len(validationErrors) > 0 {
		utils.ValidationErrorResponse(w, validationErrors)
		return
	}

	// Check if email exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		utils.ErrorResponse(w, http.StatusConflict, "EMAIL_EXISTS", "Email already registered")
		return
	}

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process password")
		return
	}

	// Set default role
	role := "customer"
	if req.Role == "seller" {
		role = "seller"
	}

	// Create user
	user := &models.User{
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Role:      role,
		Status:    "active",
	}

	if err := h.db.Create(user).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user")
		return
	}

	// Generate tokens
	tokenPair, err := h.jwtService.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate tokens")
		return
	}

	// Store refresh token
	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: h.jwtService.GetRefreshExpiry(),
	}
	h.db.Create(refreshToken)

	// Create cart for user
	cart := &models.Cart{UserID: user.ID}
	h.db.Create(cart)

	utils.JSONResponse(w, http.StatusCreated, AuthResponse{
		User:         user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email and password are required")
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Check password
	if err := auth.CheckPassword(req.Password, user.Password); err != nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Check if user is suspended
	if user.Status == "suspended" {
		utils.ErrorResponse(w, http.StatusForbidden, "ACCOUNT_SUSPENDED", "Your account has been suspended")
		return
	}

	// Generate tokens
	tokenPair, err := h.jwtService.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate tokens")
		return
	}

	// Store refresh token
	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: h.jwtService.GetRefreshExpiry(),
	}
	h.db.Create(refreshToken)

	utils.JSONResponse(w, http.StatusOK, AuthResponse{
		User:         &user,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// In a real app, you'd blacklist the token or delete the refresh token
	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": "Successfully logged out"})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required")
		return
	}

	// Validate refresh token
	claims, err := h.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired refresh token")
		return
	}

	// Check if refresh token exists and is not revoked
	var storedToken models.RefreshToken
	if err := h.db.Where("token = ? AND revoked = ?", req.RefreshToken, false).First(&storedToken).Error; err != nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", "Refresh token not found or revoked")
		return
	}

	// Get user
	userID, _ := uuid.Parse(claims.UserID)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", "User not found")
		return
	}

	// Revoke old refresh token
	h.db.Model(&storedToken).Update("revoked", true)

	// Generate new tokens
	tokenPair, err := h.jwtService.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate tokens")
		return
	}

	// Store new refresh token
	newRefreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: h.jwtService.GetRefreshExpiry(),
	}
	h.db.Create(newRefreshToken)

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"accessToken":  tokenPair.AccessToken,
		"refreshToken": tokenPair.RefreshToken,
		"tokenType":    tokenPair.TokenType,
		"expiresIn":    tokenPair.ExpiresIn,
	})
}

// GetMe returns current authenticated user
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userCtx := getUserContext(r)
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	var user models.User
	if err := h.db.Preload("Addresses").Preload("Cart.Items.Product").First(&user, userID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	utils.JSONResponse(w, http.StatusOK, user)
}

// UpdateMe updates current authenticated user
func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userCtx := getUserContext(r)
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var updateData map[string]interface{}
	if err := utils.ParseJSON(r, &updateData); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Only allow updating certain fields
	allowedFields := map[string]bool{"firstName": true, "lastName": true, "phone": true}
	for key := range updateData {
		if !allowedFields[key] {
			delete(updateData, key)
		}
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	if err := h.db.Model(&models.User{}).Where("id = ?", userID).Updates(updateData).Error; err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update user")
		return
	}

	var user models.User
	h.db.First(&user, userID)
	utils.JSONResponse(w, http.StatusOK, user)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userCtx := getUserContext(r)
	if userCtx == nil {
		utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	var req PasswordChangeRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Current and new passwords are required")
		return
	}

	if len(req.NewPassword) < 8 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "New password must be at least 8 characters")
		return
	}

	userID, _ := uuid.Parse(userCtx.UserID)
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Verify current password
	if err := auth.CheckPassword(req.CurrentPassword, user.Password); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_PASSWORD", "Current password is incorrect")
		return
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process password")
		return
	}

	// Update password
	h.db.Model(&user).Update("password", hashedPassword)

	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

// RequestPasswordReset handles password reset request
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Email == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Email is required")
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "Email not found")
		return
	}

	// Generate reset token (in production, this would be sent via email)
	resetToken := uuid.New().String()

	// For demo purposes, we'll return the token (in production, send via email)
	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message":    "Password reset token generated",
		"resetToken": resetToken,
	})
}

// ConfirmPasswordReset handles password reset confirmation
func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetConfirmRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Token and new password are required")
		return
	}

	if len(req.NewPassword) < 8 {
		utils.ErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters")
		return
	}

	// In a real app, you would validate the token and find the associated user
	// For demo purposes, we'll just return success
	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": "Password has been reset successfully"})
}

// Helper to get user context from request
func getUserContext(r *http.Request) *middleware.UserContext {
	return middleware.GetUserFromContext(r.Context())
}

// Helper for password reset tokens (in production, use proper token storage)
var passwordResetTokens = make(map[string]struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
})
