package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/rkgcloud/crud/pkg/config"
	"github.com/rkgcloud/crud/pkg/controllers"
	"github.com/rkgcloud/crud/pkg/database"
	"github.com/rkgcloud/crud/pkg/health"
	"github.com/rkgcloud/crud/pkg/middleware"
	"github.com/rkgcloud/crud/pkg/models"
	"github.com/rkgcloud/crud/pkg/session"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

const (
	templatesPath = "templates/*.html"
	sessionName   = "session"
	appVersion    = "1.0.0"
)

type App struct {
	config        *config.Config
	db            *gorm.DB
	router        *gin.Engine
	server        *http.Server
	healthChecker *health.HealthChecker
}

func main() {
	app := &App{}
	if err := app.Initialize(); err != nil {
		log.Fatal("Failed to initialize application:", err)
	}

	// Start server in a goroutine
	go func() {
		if err := app.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to run application:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	app.GracefulShutdown()
}

func (app *App) Initialize() error {
	var err error

	// Load configuration
	if app.config, err = config.Load(); err != nil {
		return fmt.Errorf("configuration loading failed: %w", err)
	}

	// Initialize database
	if app.db, err = app.initializeDB(); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}

	// Initialize health checker
	app.healthChecker = health.NewHealthChecker(app.db, appVersion)

	// Set Gin mode
	if app.config.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router with middleware
	if err = app.setupRouter(); err != nil {
		return fmt.Errorf("router setup failed: %w", err)
	}

	// Initialize HTTP server
	app.server = &http.Server{
		Addr:         ":" + app.config.Server.Port,
		Handler:      app.router,
		ReadTimeout:  app.config.Server.ReadTimeout,
		WriteTimeout: app.config.Server.WriteTimeout,
	}

	return nil
}

func (app *App) initializeDB() (*gorm.DB, error) {
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
			log.Printf("Unauthorized access attempt from IP: %s to path: %s", c.ClientIP(), c.Request.URL.Path)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		loggedInUser := session.GetLoggedInUser(c)
		if loggedInUser.ID == "" {
			log.Printf("Invalid session for IP: %s", c.ClientIP())
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("loggedInUser", loggedInUser)
		c.Next()
	}
}

func (app *App) setupRouter() error {
	// Create router without default middleware
	app.router = gin.New()

	// Add custom middleware in order
	app.router.Use(middleware.Recovery())
	app.router.Use(middleware.RequestLogger())
	app.router.Use(middleware.SecurityHeaders(app.config))
	app.router.Use(middleware.CORS(app.config))
	app.router.Use(middleware.RateLimiter(app.config))
	app.router.Use(middleware.RequestTimeout(app.config.Server.ReadTimeout))

	// Session store
	store := cookie.NewStore([]byte(app.config.Session.Secret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   app.config.Session.MaxAge,
		HttpOnly: app.config.Session.HttpOnly,
		Secure:   app.config.Session.Secure,
		SameSite: http.SameSiteStrictMode,
	})
	app.router.Use(sessions.Sessions(sessionName, store))

	// Load templates
	templateDir := os.Getenv("KO_DATA_PATH")
	if templateDir != "" {
		app.router.LoadHTMLGlob(path.Join(templateDir, templatesPath))
	} else {
		// Fallback to local kodata directory for development
		app.router.LoadHTMLGlob("./kodata/templates/*.html")
	}

	// Static files
	app.router.Static("/images", "./kodata/templates/images")

	// Health check endpoints (no auth required)
	healthGroup := app.router.Group("/health")
	{
		healthGroup.GET("/live", app.healthChecker.LivenessHandler)
		healthGroup.GET("/ready", app.healthChecker.ReadinessHandler)
		healthGroup.GET("/", app.healthChecker.HealthHandler)
		healthGroup.GET("/metrics", app.healthChecker.MetricsHandler)
	}

	// Public routes
	app.router.GET("/login", controllers.LoginPage)
	app.router.GET("/auth/google", controllers.HandleGoogleLogin)
	app.router.GET("/auth/callback", controllers.HandleGoogleCallback)
	app.router.GET("/logout", controllers.Logout)

	// Protected routes
	protected := app.router.Group("/")
	protected.Use(app.authMiddleware())
	{
		protected.GET("/", func(c *gin.Context) { controllers.Index(c, app.db) })
		protected.GET("/accounts", func(c *gin.Context) { controllers.GetAccounts(c, app.db) })
		protected.GET("/image", controllers.ImagePage)
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
	log.Printf("Starting server on port %s", app.config.Server.Port)
	return app.server.ListenAndServe()
}

// GracefulShutdown handles graceful shutdown of the server
func (app *App) GracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), app.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown the server
	if err := app.server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Close database connection
	if app.db != nil {
		sqlDB, err := app.db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Error closing database connection: %v", err)
			}
		}
	}

	log.Println("Server exited")
}
