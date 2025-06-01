package main

import (
	"log"
	"os"
	"path"

	"github.com/rkgcloud/crud/pkg/api/handlers"
	"github.com/rkgcloud/crud/pkg/database"
	"github.com/rkgcloud/crud/pkg/models"

	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to database
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	err = db.AutoMigrate(&models.User{}, &models.Account{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Set up router
	router := gin.Default()
	templateDir := os.Getenv("KO_DATA_PATH")
	router.LoadHTMLGlob(path.Join(templateDir, "templates/*.html"))

	// Serve the HTML page
	router.GET("/", func(c *gin.Context) { handlers.Index(c, db) })

	// Define routes
	router.POST("/users", func(c *gin.Context) { handlers.CreateUser(c, db) })
	router.GET("/users", func(c *gin.Context) { handlers.GetUsers(c, db) })
	router.GET("/users/:id", func(c *gin.Context) { handlers.GetUser(c, db) })
	router.PUT("/users/:id", func(c *gin.Context) { handlers.UpdateUser(c, db) })
	router.DELETE("/users/:id", func(c *gin.Context) { handlers.DeleteUser(c, db) })
	router.POST("/accounts", func(c *gin.Context) { handlers.CreateAccount(c, db) })
	router.GET("/accounts", func(c *gin.Context) { handlers.GetAccounts(c, db) })

	// Run server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
