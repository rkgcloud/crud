package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/rkgcloud/crud/pkg/auth"
	"github.com/rkgcloud/crud/pkg/controllers"
	"github.com/rkgcloud/crud/pkg/database"
	"github.com/rkgcloud/crud/pkg/models"
	"github.com/rkgcloud/crud/pkg/session"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort   = "8080"
	templatesPath = "templates/*.html"
)

type App struct {
	db           *gorm.DB
	router       *gin.Engine
	loggedInUser *auth.LoggedInUser
}

func main() {
	app := &App{}
	if err := app.Initialize(); err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	if err := app.Run(); err != nil {
		log.Fatal("Failed to run application:", err)
	}
}

func (app *App) Initialize() error {
	var err error

	// Initialize database
	if app.db, err = initializeDB(); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}

	// Session store
	secret := os.Getenv("SECRET")
	if secret == "" {
		secret = "secret" // only used in local dev
	}
	store := cookie.NewStore([]byte(secret))

	debug := os.Getenv("DEBUG")
	if debug == "true" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	app.router = gin.Default()
	app.router.Use(sessions.Sessions("session", store))
	if err = app.setupRoutes(); err != nil {
		return fmt.Errorf("route setup failed: %w", err)
	}

	return nil
}

func initializeDB() (*gorm.DB, error) {
	db, err := database.ConnectDB()
	if err != nil {
		return nil, err
	}

	if err = db.AutoMigrate(&models.User{}, &models.Account{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func (app *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !session.IsLoggedIn(c) {
			log.Println("User not logged in")
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Set("loggedInUser", session.GetLoggedInUser(c))
		c.Next()
	}
}

func (app *App) setupRoutes() error {

	// Load templates
	templateDir := os.Getenv("KO_DATA_PATH")
	app.router.LoadHTMLGlob(path.Join(templateDir, templatesPath))

	// static images
	app.router.Static("/images", "./kodata/templates/images")

	// Public routes
	app.router.GET("/login", controllers.LoginPage)
	app.router.GET("/auth/google", controllers.HandleGoogleLogin)
	app.router.GET("/auth/callback", controllers.HandleGoogleCallback)
	app.router.GET("/logout", controllers.Logout)

	// Default routes group
	defaultRoutes := app.router.Group("/")
	defaultRoutes.Use(app.authMiddleware())
	{
		defaultRoutes.GET("/", func(c *gin.Context) { controllers.Index(c, app.db) })
	}

	// User routes group
	userRoutes := app.router.Group("/users")
	userRoutes.Use(app.authMiddleware())
	{
		userRoutes.POST("/", func(c *gin.Context) { controllers.CreateUser(c, app.db) })
		userRoutes.GET("/", func(c *gin.Context) { controllers.GetUsers(c, app.db) })
		userRoutes.GET("/:id", func(c *gin.Context) { controllers.GetUser(c, app.db) })
		userRoutes.PUT("/:id", func(c *gin.Context) { controllers.UpdateUser(c, app.db) })
		userRoutes.DELETE("/:id", func(c *gin.Context) { controllers.DeleteUser(c, app.db) })
	}

	// Account routes group
	accountRoutes := app.router.Group("/accounts")
	accountRoutes.Use(app.authMiddleware())
	{
		accountRoutes.POST("/", func(c *gin.Context) { controllers.CreateAccount(c, app.db) })
		accountRoutes.GET("/", func(c *gin.Context) { controllers.GetAccounts(c, app.db) })
		accountRoutes.POST("/update/:id", func(c *gin.Context) { controllers.UpdateAccount(c, app.db) })
	}

	return nil
}

func (app *App) Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return app.router.Run(":" + port)
}
