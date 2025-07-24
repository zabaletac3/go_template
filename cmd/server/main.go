package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-template/internal/container"
	"go-template/internal/database"
	"go-template/internal/shared/response"
)

// @title Go API Template
// @version 1.0
// @description A robust, scalable Go API template with dependency container architecture
// @host localhost:8080
// @BasePath /api/v1
func main() {
	log.Println("🚀 Starting Go API Template Server...")

	// Create dependency container
	deps := container.NewDependencies()

	// Initialize all dependencies
	if err := deps.Initialize(); err != nil {
		log.Fatalf("❌ Failed to initialize dependencies: %v", err)
	}

	// Setup basic test routes for Phase 1
	setupTestRoutes(deps)

	// Create HTTP server with optimized settings
	server := &http.Server{
		Addr:         deps.GetConfig().GetServerAddress(),
		Handler:      deps.Mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger := deps.GetLogger("server")
		logger.Info("🌟 Server starting", "port", deps.GetConfig().Port, "env", deps.GetConfig().Environment)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	}

	// Close all dependencies
	if err := deps.Close(); err != nil {
		log.Printf("⚠️  Error closing dependencies: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}

// setupTestRoutes sets up basic test routes for Phase 1 validation
func setupTestRoutes(deps *container.Dependencies) {
	logger := deps.GetLogger("routes")
	logger.Info("Setting up test routes for Phase 1")

	mux := deps.Mux

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Health check requested")
		
		health := map[string]interface{}{
			"status":      "healthy",
			"version":     "1.0.0",
			"environment": deps.GetConfig().Environment,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}

		// Check database connection
		if err := database.PingMongoDB(deps.GetDB()); err != nil {
			health["database"] = "unhealthy"
			health["database_error"] = err.Error()
			logger.Error("Database health check failed", err)
			response.ErrorWithDetails(w, "HEALTH_CHECK_FAILED", "Database is unhealthy", health, http.StatusServiceUnavailable)
			return
		}
		health["database"] = "healthy"

		// Check Redis connection
		if err := deps.GetCache().Ping(r.Context()); err != nil {
			health["cache"] = "unhealthy"
			health["cache_error"] = err.Error()
			logger.Error("Cache health check failed", err)
			response.ErrorWithDetails(w, "HEALTH_CHECK_FAILED", "Cache is unhealthy", health, http.StatusServiceUnavailable)
			return
		}
		health["cache"] = "healthy"

		response.JSON(w, health, http.StatusOK)
	})

	// Database test endpoint
	mux.HandleFunc("GET /test/database", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Database test requested")
		
		// Test database connection
		collections, err := database.GetCollectionNames(deps.GetDB())
		if err != nil {
			logger.Error("Failed to get collection names", err)
			response.InternalServerError(w)
			return
		}

		testData := map[string]interface{}{
			"message":     "Database connection successful",
			"database":    deps.GetConfig().DatabaseName,
			"collections": collections,
		}

		response.JSONWithMessage(w, testData, "Database test passed", http.StatusOK)
	})

	// Cache test endpoint
	mux.HandleFunc("GET /test/cache", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Cache test requested")
		
		ctx := r.Context()
		testKey := "test:cache:key"
		testValue := "Hello, Redis!"

		// Test cache set
		if err := deps.GetCache().Set(ctx, testKey, testValue, 5*time.Minute); err != nil {
			logger.Error("Failed to set cache value", err)
			response.InternalServerError(w)
			return
		}

		// Test cache get
		retrievedValue, err := deps.GetCache().Get(ctx, testKey)
		if err != nil {
			logger.Error("Failed to get cache value", err)
			response.InternalServerError(w)
			return
		}

		testData := map[string]interface{}{
			"message":        "Cache connection successful",
			"test_key":       testKey,
			"test_value":     testValue,
			"retrieved_value": retrievedValue,
			"values_match":   testValue == retrievedValue,
		}

		response.JSONWithMessage(w, testData, "Cache test passed", http.StatusOK)
	})

	// Configuration test endpoint
	mux.HandleFunc("GET /test/config", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Configuration test requested")
		
		config := deps.GetConfig()
		testData := map[string]interface{}{
			"message":     "Configuration loaded successfully",
			"port":        config.Port,
			"environment": config.Environment,
			"log_level":   config.LogLevel,
			"database":    config.DatabaseName,
			"is_dev":      config.IsDevelopment(),
			"is_prod":     config.IsProduction(),
			"is_test":     config.IsTest(),
		}

		response.JSONWithMessage(w, testData, "Configuration test passed", http.StatusOK)
	})

	// JSON response test endpoint
	mux.HandleFunc("GET /test/responses", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Response formats test requested")
		
		// Test different response formats based on query parameter
		format := r.URL.Query().Get("format")
		
		switch format {
		case "error":
			response.Error(w, "This is a test error", http.StatusBadRequest)
		case "validation":
			validationErrors := []response.ValidationError{
				response.NewValidationError("email", "Email is required", ""),
				response.NewValidationError("password", "Password must be at least 8 characters", "123"),
			}
			response.ValidationErrors(w, validationErrors)
		case "not_found":
			response.NotFound(w, "Test resource")
		case "unauthorized":
			response.Unauthorized(w, "Invalid credentials")
		case "created":
			response.Created(w, map[string]string{"id": "123", "name": "Test"}, "")
		default:
			testData := map[string]interface{}{
				"message": "Response system working correctly",
				"available_formats": []string{
					"?format=error",
					"?format=validation", 
					"?format=not_found",
					"?format=unauthorized",
					"?format=created",
				},
			}
			response.JSON(w, testData, http.StatusOK)
		}
	})

	// Root endpoint
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Root endpoint accessed")
		
		welcomeData := map[string]interface{}{
			"message":     "🚀 Welcome to Go API Template",
			"version":     "1.0.0",
			"environment": deps.GetConfig().Environment,
			"endpoints": map[string]string{
				"health":      "/health",
				"db_test":     "/test/database", 
				"cache_test":  "/test/cache",
				"config_test": "/test/config",
				"response_test": "/test/responses",
			},
		}

		response.JSONWithMessage(w, welcomeData, "Phase 1 - Infrastructure ready!", http.StatusOK)
	})

	logger.Info("✅ Test routes setup completed")
}