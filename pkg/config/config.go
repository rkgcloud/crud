package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Session  SessionConfig
	OAuth    OAuthConfig
	Security SecurityConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port            string
	Debug           bool
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	URL             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// SessionConfig holds session-related configuration
type SessionConfig struct {
	Secret   string
	MaxAge   int
	HttpOnly bool
	Secure   bool
	SameSite string
}

// OAuthConfig holds OAuth-related configuration
type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	RedirectURL        string
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimitPerMinute int
	RateLimitBurst     int
	AllowedOrigins     []string
	CSPPolicy          string
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			Debug:           getEnvBool("DEBUG", false),
			ReadTimeout:     getEnvDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=testdb port=5432 sslmode=disable"),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		},
		Session: SessionConfig{
			Secret:   getEnv("SECRET", ""),
			MaxAge:   getEnvInt("SESSION_MAX_AGE", 86400*7), // 7 days
			HttpOnly: getEnvBool("SESSION_HTTP_ONLY", true),
			Secure:   getEnvBool("SESSION_SECURE", !getEnvBool("DEBUG", false)),
			SameSite: getEnv("SESSION_SAME_SITE", "Strict"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:        getEnv("OAUTH_REDIRECT_URL", "http://localhost:8080/auth/callback"),
		},
		Security: SecurityConfig{
			RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 60),
			RateLimitBurst:     getEnvInt("RATE_LIMIT_BURST", 10),
			AllowedOrigins:     getEnvSlice("ALLOWED_ORIGINS", []string{"http://localhost:8080"}),
			CSPPolicy:          getEnv("CSP_POLICY", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'"),
		},
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.Session.Secret == "" {
		log.Printf("WARNING: SESSION SECRET not set. Using default secret for development only")
		c.Session.Secret = "change-me-in-production-this-is-not-secure"
	}

	if len(c.Session.Secret) < 32 {
		return fmt.Errorf("session secret must be at least 32 characters long, got %d", len(c.Session.Secret))
	}

	if c.OAuth.GoogleClientID == "" {
		log.Printf("WARNING: GOOGLE_CLIENT_ID not set. OAuth authentication will not work")
	}

	if c.OAuth.GoogleClientSecret == "" {
		log.Printf("WARNING: GOOGLE_CLIENT_SECRET not set. OAuth authentication will not work")
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
		log.Printf("Invalid boolean value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
		log.Printf("Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
		log.Printf("Invalid duration value for %s: %s, using default: %v", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		result := []string{}
		for _, item := range []string{value} {
			if trimmed := item; trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
