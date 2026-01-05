package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketplace-api/internal/auth"
	"github.com/marketplace-api/internal/config"
	"github.com/marketplace-api/internal/database"
	"github.com/marketplace-api/internal/router"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations if enabled
	if cfg.AutoMigrate {
		if err := db.AutoMigrate(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("✅ Database migrations completed")
	} else {
		log.Println("⏭️  Auto-migration disabled, skipping...")
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg)

	// Setup router
	r := router.New(db, jwtService)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on port %s", cfg.Port)
		log.Printf("📚 API Documentation: http://localhost:%s/api/v1", cfg.Port)
		log.Printf("❤️  Health Check: http://localhost:%s/health", cfg.Port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}
