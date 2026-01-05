package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/marketplace-api/internal/auth"
	"github.com/marketplace-api/internal/utils"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

type UserContext struct {
	UserID string
	Email  string
	Role   string
}

type AuthMiddleware struct {
	jwtService *auth.JWTService
}

func NewAuthMiddleware(jwtService *auth.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtService}
}

// RequireAuth middleware that requires valid JWT token
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}

		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			if err == auth.ErrExpiredToken {
				utils.ErrorResponse(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
				return
			}
			utils.ErrorResponse(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid authentication token")
			return
		}

		// Add user info to context
		userCtx := &UserContext{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		}
		ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware that checks for specific roles
func (m *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userCtx := GetUserFromContext(r.Context())
			if userCtx == nil {
				utils.ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}

			hasRole := false
			for _, role := range roles {
				if userCtx.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				utils.ErrorResponse(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth middleware that extracts user if token is present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			claims, err := m.jwtService.ValidateToken(token)
			if err == nil {
				userCtx := &UserContext{
					UserID: claims.UserID,
					Email:  claims.Email,
					Role:   claims.Role,
				}
				ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}
	return ""
}

// GetUserFromContext extracts user context from request context
func GetUserFromContext(ctx context.Context) *UserContext {
	userCtx, ok := ctx.Value(UserContextKey).(*UserContext)
	if !ok {
		return nil
	}
	return userCtx
}
