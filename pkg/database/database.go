package database

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectDB connects to the PostgresSQL database
func ConnectDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Println("DATABASE_URL not set, using default local connection")
		dsn = "host=localhost user=postgres password=postgres dbname=testdb port=5432 sslmode=disable"
	}

	// Don't log the full connection string as it may contain sensitive information
	log.Println("Connecting to database...")

	// Configure GORM with better settings
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // Only log errors in production
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt:                              true,  // Cache prepared statements
		DisableForeignKeyConstraintWhenMigrating: false, // Keep foreign key constraints
	}

	// Set debug mode based on environment
	if os.Getenv("DEBUG") == "true" {
		config.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get underlying sql.DB: %v", err)
		return nil, err
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connected successfully")
	return db, nil
}
