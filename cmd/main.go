package main

import (
	"log"
	"os"

	"github.com/rkgcloud/crud/pkg/api/handlers"
	"github.com/rkgcloud/crud/pkg/database"
	models "github.com/rkgcloud/crud/pkg/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to database
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Set up router
	r := gin.Default()

	// Define routes
	r.POST("/users", func(c *gin.Context) { handlers.CreateUser(c, db) })
	r.GET("/users", func(c *gin.Context) { handlers.GetUsers(c, db) })
	r.GET("/users/:id", func(c *gin.Context) { handlers.GetUser(c, db) })
	r.PUT("/users/:id", func(c *gin.Context) { handlers.UpdateUser(c, db) })
	r.DELETE("/users/:id", func(c *gin.Context) { handlers.DeleteUser(c, db) })

	// Run server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
