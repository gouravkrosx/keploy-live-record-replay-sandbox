package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/marketplace-api/internal/config"
	"github.com/marketplace-api/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	*gorm.DB
	sqlDB *sql.DB
}

func New(cfg *config.Config) (*Database, error) {
	dsn := cfg.GetDSN()

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.Env == "production" {
		gormLogger = logger.Default.LogMode(logger.Warn)
	}

	// Open connection
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                 gormLogger,
		PrepareStmt:            true, // Enable prepared statement cache
		SkipDefaultTransaction: true, // Disable default transaction for better performance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Ping database to verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Database connected successfully")

	return &Database{DB: db, sqlDB: sqlDB}, nil
}

// Ping checks database connectivity
func (d *Database) Ping() error {
	return d.sqlDB.Ping()
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.sqlDB.Close()
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	log.Println("🔄 Running database migrations...")

	err := d.DB.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
		&models.Review{},
		&models.Address{},
		&models.Payment{},
		&models.Cart{},
		&models.CartItem{},
		&models.Inventory{},
		&models.Coupon{},
		&models.RefreshToken{},
	)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// RawQuery executes a raw SQL query (for demonstration of normal queries)
func (d *Database) RawQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return d.sqlDB.Query(query, args...)
}

// PreparedStatement demonstrates prepared statement usage
func (d *Database) PreparedStatement(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := d.sqlDB.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Query(args...)
}
