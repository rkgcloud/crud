package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectDB connects to the PostgresSQL database
func ConnectDB() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=testdb port=5432 sslmode=disable"
	}
	log.Printf("connection string %q\n", dsn)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("failed to connect database: %v\n", err)
		return nil, err
	}
	log.Println("Database connected successfully")
	return db, nil
}
