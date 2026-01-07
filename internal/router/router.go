package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/marketplace-api/internal/auth"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/handlers"
	authMiddleware "github.com/marketplace-api/internal/middleware"
)

func New(db *database.Database, jwtService *auth.JWTService) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize auth middleware
	authMw := authMiddleware.NewAuthMiddleware(jwtService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, jwtService)
	userHandler := handlers.NewUserHandler(db)
	productHandler := handlers.NewProductHandler(db)
	categoryHandler := handlers.NewCategoryHandler(db)
	orderHandler := handlers.NewOrderHandler(db)
	cartHandler := handlers.NewCartHandler(db)
	inventoryHandler := handlers.NewInventoryHandler(db)
	couponHandler := handlers.NewCouponHandler(db)
	analyticsHandler := handlers.NewAnalyticsHandler(db)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"disconnected"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","database":"connected"}`))
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// ==================== AUTHENTICATION ====================
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/password/reset", authHandler.RequestPasswordReset)
			r.Post("/password/reset/confirm", authHandler.ConfirmPasswordReset)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(authMw.RequireAuth)
				r.Post("/logout", authHandler.Logout)
				r.Get("/me", authHandler.GetMe)
				r.Put("/me", authHandler.UpdateMe)
				r.Put("/password", authHandler.ChangePassword)
			})
		})

		// ==================== USERS ====================
		r.Route("/users", func(r chi.Router) {
			r.Use(authMw.RequireAuth)

			r.Get("/", userHandler.ListUsers)
			r.Post("/", userHandler.CreateUser)

			r.Route("/{userId}", func(r chi.Router) {
				r.Get("/", userHandler.GetUser)
				r.Put("/", userHandler.UpdateUser)
				r.Delete("/", userHandler.DeleteUser)
				r.Get("/orders", userHandler.GetUserOrders)
				r.Get("/addresses", userHandler.ListUserAddresses)
				r.Post("/addresses", userHandler.CreateUserAddress)
			})
		})

		// ==================== PRODUCTS ====================
		r.Route("/products", func(r chi.Router) {
			// Public routes
			r.Get("/", productHandler.ListProducts)
			r.Get("/{productId}", productHandler.GetProduct)
			r.Get("/{productId}/reviews", productHandler.GetProductReviews)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(authMw.RequireAuth)
				r.Post("/", productHandler.CreateProduct)
				r.Post("/bulk", productHandler.BulkCreateProducts)
				r.Put("/{productId}", productHandler.UpdateProduct)
				r.Delete("/{productId}", productHandler.DeleteProduct)
				r.Post("/{productId}/reviews", productHandler.CreateProductReview)
			})
		})

		// ==================== CATEGORIES ====================
		r.Route("/categories", func(r chi.Router) {
			// Public routes
			r.Get("/", categoryHandler.ListCategories)
			r.Get("/{categoryId}", categoryHandler.GetCategory)

			// Protected routes (admin)
			r.Group(func(r chi.Router) {
				r.Use(authMw.RequireAuth)
				r.Use(authMw.RequireRole("admin", "seller"))
				r.Post("/", categoryHandler.CreateCategory)
				r.Put("/{categoryId}", categoryHandler.UpdateCategory)
				r.Delete("/{categoryId}", categoryHandler.DeleteCategory)
			})
		})

		// ==================== ORDERS ====================
		r.Route("/orders", func(r chi.Router) {
			r.Use(authMw.RequireAuth)

			r.Get("/", orderHandler.ListOrders)
			r.Post("/", orderHandler.CreateOrder)

			r.Route("/{orderId}", func(r chi.Router) {
				r.Get("/", orderHandler.GetOrder)
				r.Put("/status", orderHandler.UpdateOrderStatus)
				r.Post("/cancel", orderHandler.CancelOrder)
				r.Post("/refund", orderHandler.RefundOrder)
			})
		})

		// ==================== CART ====================
		r.Route("/cart", func(r chi.Router) {
			r.Use(authMw.RequireAuth)

			r.Get("/", cartHandler.GetCart)
			r.Delete("/", cartHandler.ClearCart)
			r.Post("/items", cartHandler.AddCartItem)
			r.Put("/items/{itemId}", cartHandler.UpdateCartItem)
			r.Delete("/items/{itemId}", cartHandler.RemoveCartItem)
			r.Post("/validate", cartHandler.ValidateCart)
			r.Post("/apply-coupon", cartHandler.ApplyCoupon)
		})

		// ==================== INVENTORY ====================
		r.Route("/inventory", func(r chi.Router) {
			r.Use(authMw.RequireAuth)
			r.Use(authMw.RequireRole("admin", "seller"))

			r.Get("/", inventoryHandler.ListInventory)
			r.Put("/{inventoryId}", inventoryHandler.UpdateInventory)
			r.Post("/transfer", inventoryHandler.TransferInventory)
			r.Post("/adjustments", inventoryHandler.AdjustInventory)
		})

		// ==================== COUPONS ====================
		r.Route("/coupons", func(r chi.Router) {
			// Public validation
			r.Post("/validate", couponHandler.ValidateCoupon)

			// Protected routes (admin)
			r.Group(func(r chi.Router) {
				r.Use(authMw.RequireAuth)
				r.Use(authMw.RequireRole("admin"))
				r.Get("/", couponHandler.ListCoupons)
				r.Post("/", couponHandler.CreateCoupon)
				r.Get("/{couponId}", couponHandler.GetCoupon)
				r.Put("/{couponId}", couponHandler.UpdateCoupon)
				r.Delete("/{couponId}", couponHandler.DeleteCoupon)
			})
		})

		// ==================== ANALYTICS ====================
		r.Route("/analytics", func(r chi.Router) {
			r.Use(authMw.RequireAuth)
			r.Use(authMw.RequireRole("admin", "seller"))

			r.Get("/sales", analyticsHandler.GetSalesAnalytics)
			r.Get("/products/bestsellers", analyticsHandler.GetBestsellers)
			r.Get("/customers", analyticsHandler.GetCustomerAnalytics)
			r.Get("/inventory/alerts", analyticsHandler.GetInventoryAlerts)
		})

		// for large size mocks
		// ==================== HEAVY DATA (for testing large database operations) ====================
		heavyHandler := handlers.NewHeavyHandler(db)
		r.Route("/heavy", func(r chi.Router) {
			r.Use(authMw.RequireAuth)

			r.Get("/products", heavyHandler.HeavyProducts)     // for large size mocks
			r.Get("/orders", heavyHandler.HeavyOrders)         // for large size mocks
			r.Get("/reviews", heavyHandler.HeavyReviews)       // for large size mocks
			r.Get("/inventory", heavyHandler.HeavyInventory)   // for large size mocks
			r.Get("/aggregate", heavyHandler.HeavyAggregate)   // for large size mocks
			r.Get("/users", heavyHandler.HeavyUsers)           // for large size mocks
			r.Get("/categories", heavyHandler.HeavyCategories) // for large size mocks
			r.Get("/payments", heavyHandler.HeavyPayments)     // for large size mocks
			r.Get("/carts", heavyHandler.HeavyCarts)           // for large size mocks
			r.Get("/full-dump", heavyHandler.HeavyFullDump)    // for large size mocks
			// New heavy endpoints
			r.Get("/product-search", heavyHandler.HeavyProductSearch)           // for large size mocks
			r.Get("/order-history", heavyHandler.HeavyOrderHistory)             // for large size mocks
			r.Get("/user-activity", heavyHandler.HeavyUserActivity)             // for large size mocks
			r.Get("/analytics-dashboard", heavyHandler.HeavyAnalyticsDashboard) // for large size mocks
			r.Get("/sales-trends", heavyHandler.HeavySalesTrends)               // for large size mocks
			r.Get("/inventory-report", heavyHandler.HeavyInventoryReport)       // for large size mocks
			r.Get("/review-sentiment", heavyHandler.HeavyReviewSentiment)       // for large size mocks
			r.Get("/category-tree", heavyHandler.HeavyCategoryTree)             // for large size mocks
			r.Get("/shipping-data", heavyHandler.HeavyShippingData)             // for large size mocks
			r.Get("/financial-summary", heavyHandler.HeavyFinancialSummary)     // for large size mocks
		})
	})

	return r
}
