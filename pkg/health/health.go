package health

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version,omitempty"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Status  string        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency,omitempty"`
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	db      *gorm.DB
	version string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *gorm.DB, version string) *HealthChecker {
	return &HealthChecker{
		db:      db,
		version: version,
	}
}

// LivenessHandler handles liveness probe requests
func (h *HealthChecker) LivenessHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	})
}

// ReadinessHandler handles readiness probe requests
func (h *HealthChecker) ReadinessHandler(c *gin.Context) {
	checks := make(map[string]CheckResult)
	overallStatus := "ready"

	// Database connectivity check
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "not_ready"
	}

	status := http.StatusOK
	if overallStatus != "ready" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC(),
		Version:   h.version,
		Checks:    checks,
	})
}

// HealthHandler handles comprehensive health check requests
func (h *HealthChecker) HealthHandler(c *gin.Context) {
	checks := make(map[string]CheckResult)
	overallStatus := "healthy"

	// Database connectivity check
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Memory usage check
	memCheck := h.checkMemory()
	checks["memory"] = memCheck
	if memCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	status := http.StatusOK
	if overallStatus == "unhealthy" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now().UTC(),
		Version:   h.version,
		Checks:    checks,
	})
}

// MetricsHandler provides basic application metrics
func (h *HealthChecker) MetricsHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sqlDB, err := h.db.DB()
	var dbStats interface{}
	if err == nil {
		dbStats = sqlDB.Stats()
	}

	metrics := gin.H{
		"timestamp": time.Now().UTC(),
		"memory": gin.H{
			"alloc_bytes":       m.Alloc,
			"total_alloc_bytes": m.TotalAlloc,
			"sys_bytes":         m.Sys,
			"gc_cycles":         m.NumGC,
			"goroutines":        runtime.NumGoroutine(),
		},
		"database": dbStats,
		"runtime": gin.H{
			"go_version":    runtime.Version(),
			"num_cpu":       runtime.NumCPU(),
			"num_goroutine": runtime.NumGoroutine(),
		},
	}

	c.JSON(http.StatusOK, metrics)
}

// checkDatabase verifies database connectivity
func (h *HealthChecker) checkDatabase() CheckResult {
	start := time.Now()

	if h.db == nil {
		return CheckResult{
			Status:  "unhealthy",
			Message: "Database connection is nil",
			Latency: time.Since(start),
		}
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		return CheckResult{
			Status:  "unhealthy",
			Message: "Failed to get underlying database connection",
			Latency: time.Since(start),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return CheckResult{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
			Latency: time.Since(start),
		}
	}

	return CheckResult{
		Status:  "healthy",
		Message: "Database connection is healthy",
		Latency: time.Since(start),
	}
}

// checkMemory performs basic memory usage checks
func (h *HealthChecker) checkMemory() CheckResult {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB for easier reading
	allocMB := m.Alloc / 1024 / 1024
	sysMB := m.Sys / 1024 / 1024

	// Simple threshold checks (these can be made configurable)
	const (
		allocWarningThresholdMB = 512  // 512MB
		sysWarningThresholdMB   = 1024 // 1GB
	)

	status := "healthy"
	message := "Memory usage is within normal limits"

	if allocMB > allocWarningThresholdMB || sysMB > sysWarningThresholdMB {
		status = "degraded"
		message = "Memory usage is elevated"
	}

	return CheckResult{
		Status:  status,
		Message: message,
	}
}
