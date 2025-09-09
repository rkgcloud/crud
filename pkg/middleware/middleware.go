package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rkgcloud/crud/pkg/config"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimiter creates a rate limiting middleware
func RateLimiter(cfg *config.Config) gin.HandlerFunc {
	// Create rate limiter with memory store
	rate := limiter.Rate{
		Period: time.Minute,
		Limit:  int64(cfg.Security.RateLimitPerMinute),
	}

	store := memory.NewStore()
	rateLimiter := limiter.New(store, rate)

	return func(c *gin.Context) {
		// Use client IP as the key for rate limiting
		key := c.ClientIP()

		limiterContext, err := rateLimiter.Get(c.Request.Context(), key)
		if err != nil {
			log.Printf("Rate limiter error for IP %s: %v", key, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiterContext.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limiterContext.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limiterContext.Reset))

		if limiterContext.Reached {
			log.Printf("Rate limit exceeded for IP: %s", key)
			resetTime := time.Unix(limiterContext.Reset, 0)
			retryAfter := time.Until(resetTime).Seconds()
			if retryAfter < 0 {
				retryAfter = 0
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CORS creates a CORS middleware with secure defaults
func CORS(cfg *config.Config) gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowOrigins:     cfg.Security.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// In development, allow all origins if not explicitly set
	if cfg.Server.Debug && len(cfg.Security.AllowedOrigins) == 1 && cfg.Security.AllowedOrigins[0] == "http://localhost:8080" {
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowOrigins = nil
	}

	return cors.New(corsConfig)
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		c.Header("Content-Security-Policy", cfg.Security.CSPPolicy)

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS Protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// HSTS for HTTPS connections
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Prevent information disclosure
		c.Header("X-Powered-By", "")
		c.Header("Server", "")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// RequestTimeout creates a timeout middleware
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// Replace the request context
		c.Request = c.Request.WithContext(ctx)

		// Channel to signal when the request is done
		done := make(chan bool, 1)

		go func() {
			c.Next()
			done <- true
		}()

		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Request timed out
			log.Printf("Request timeout for %s from IP: %s", c.Request.URL.Path, c.ClientIP())
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
			})
			c.Abort()
			return
		}
	}
}

// RequestLogger creates a structured request logging middleware
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.ClientIP,
			param.Method,
			param.StatusCode,
			param.Latency,
			param.Path,
			param.ErrorMessage,
		)
	})
}

// Recovery creates a recovery middleware with proper logging
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Printf("Panic recovered: %v for request %s %s from IP: %s",
			recovered, c.Request.Method, c.Request.URL.Path, c.ClientIP())

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		c.Abort()
	})
}
