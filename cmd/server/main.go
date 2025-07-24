// cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "go-template/docs" // Import generated docs

	"go-template/internal/container"
	"go-template/internal/database"
	"go-template/internal/modules/users"
	"go-template/internal/shared/response"
)

// @title Go API Template
// @version 1.0
// @description A robust, scalable Go API template with Users module, dependency container architecture, MongoDB persistence, Redis caching, and comprehensive documentation.
// @termsOfService https://example.com/terms/

// @contact.name API Support
// @contact.url https://example.com/support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Users
// @tag.description User management operations including CRUD, search, and account management

// @tag.name System
// @tag.description System health and configuration endpoints

func main() {
	log.Println("🚀 Starting Go API Template Server...")

	// Create dependency container
	deps := container.NewDependencies()

	// Initialize all dependencies
	if err := deps.Initialize(); err != nil {
		log.Fatalf("❌ Failed to initialize dependencies: %v", err)
	}

	// Setup routes (Phase 1 + Phase 2 + Swagger)
	setupAllRoutes(deps)

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
		logger.Info("🌟 Server starting", 
			"port", deps.GetConfig().Port, 
			"env", deps.GetConfig().Environment,
			"version", "1.0.0",
			"swagger_ui", "http://localhost:"+deps.GetConfig().Port+"/swagger/")
		
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

// setupAllRoutes configures all application routes including Swagger
func setupAllRoutes(deps *container.Dependencies) {
	logger := deps.GetLogger("routes")
	logger.Info("🛤️  Setting up all application routes")

	// Swagger UI endpoint (FIRST - before other routes)
	setupSwaggerRoutes(deps)

	// Phase 1: Test routes (keep for debugging)
	setupTestRoutes(deps)

	// Phase 2: Business modules
	setupBusinessRoutes(deps)

	logger.Info("✅ All routes configured successfully")
}

// setupSwaggerRoutes configures Swagger UI and API documentation
func setupSwaggerRoutes(deps *container.Dependencies) {
	logger := deps.GetLogger("swagger")
	logger.Info("📚 Setting up Swagger documentation")

	mux := deps.Mux

	// Swagger UI endpoint
	mux.Handle("/swagger/", httpSwagger.WrapHandler)
	
	// API documentation info endpoint
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusPermanentRedirect)
	})

	// OpenAPI specification endpoint
	mux.HandleFunc("GET /api/v1/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("OpenAPI specification requested")
		
		openAPIInfo := map[string]interface{}{
			"message": "OpenAPI 3.0 specification available at Swagger UI",
			"swagger_ui": "/swagger/",
			"endpoints_documented": []string{
				"GET /api/v1/users",
				"POST /api/v1/users", 
				"GET /api/v1/users/{id}",
				"PUT /api/v1/users/{id}",
				"DELETE /api/v1/users/{id}",
				"GET /api/v1/users/search",
				"GET /api/v1/users/stats",
				"GET /api/v1/users/{id}/profile",
				"PUT /api/v1/users/{id}/password",
				"PUT /api/v1/users/{id}/verify",
			},
			"models_documented": []string{
				"CreateUserRequest",
				"UpdateUserRequest", 
				"ChangePasswordRequest",
				"UserResponse",
				"UserProfileResponse",
				"UserListResponse",
			},
		}

		response.JSON(w, openAPIInfo, http.StatusOK)
	})

	logger.Info("✅ Swagger documentation configured", 
		"swagger_ui", "/swagger/", 
		"api_spec", "/api/v1/openapi.json")
}

// setupBusinessRoutes registers all business logic modules
func setupBusinessRoutes(deps *container.Dependencies) {
	logger := deps.GetLogger("business")
	logger.Info("Registering business modules")

	// Users module - completely self-contained
	users.RegisterRoutes(deps)

	// Future modules will be added here:
	// products.RegisterRoutes(deps)
	// orders.RegisterRoutes(deps)
	// auth.RegisterRoutes(deps)

	logger.Info("✅ Business modules registered successfully")
}

// setupTestRoutes sets up test and system routes (from Phase 1)
func setupTestRoutes(deps *container.Dependencies) {
	logger := deps.GetLogger("system")
	logger.Info("Setting up system routes")

	mux := deps.Mux

	// Health check endpoint - Enhanced for Phase 2 + Swagger
	// @Summary System health check
	// @Description Get system health status including database and cache connectivity
	// @Tags System
	// @Accept json
	// @Produce json
	// @Success 200 {object} response.Response{data=object} "System is healthy"
	// @Failure 503 {object} response.Response{error=response.ErrorInfo} "System is unhealthy"
	// @Router /health [get]
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Health check requested")
		
		health := map[string]interface{}{
			"status":      "healthy",
			"version":     "1.0.0",
			"phase":       "2", // Updated to Phase 2
			"environment": deps.GetConfig().Environment,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"features": map[string]bool{
				"users_module":     true,
				"swagger_docs":     true,
				"mongodb":          true,
				"redis_cache":      true,
				"structured_logs":  true,
				"api_responses":    true,
			},
			"documentation": map[string]string{
				"swagger_ui":       "/swagger/",
				"api_info":         "/api/v1",
				"openapi_spec":     "/api/v1/openapi.json",
			},
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

	// API Info endpoint - Updated for Swagger
	// @Summary API information
	// @Description Get API information including available endpoints and documentation
	// @Tags System
	// @Accept json
	// @Produce json
	// @Success 200 {object} response.Response{data=object} "API information"
	// @Router /api/v1 [get]
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("API info requested")
		
		apiInfo := map[string]interface{}{
			"name":        "Go API Template",
			"version":     "1.0.0",
			"phase":       "2",
			"description": "A robust, scalable Go API template with Users module and Swagger documentation",
			"documentation": map[string]interface{}{
				"swagger_ui":     "/swagger/",
				"openapi_spec":   "/api/v1/openapi.json",
				"interactive":    "Visit /swagger/ to test the API interactively",
			},
			"endpoints": map[string]interface{}{
				"health": "/health",
				"api_info": "/api/v1",
				"users": map[string]interface{}{
					"list":         "GET /api/v1/users",
					"get":          "GET /api/v1/users/{id}",
					"create":       "POST /api/v1/users",
					"update":       "PUT /api/v1/users/{id}",
					"delete":       "DELETE /api/v1/users/{id}",
					"search":       "GET /api/v1/users/search",
					"stats":        "GET /api/v1/users/stats",
					"profile":      "GET /api/v1/users/{id}/profile",
					"change_password": "PUT /api/v1/users/{id}/password",
					"verify":       "PUT /api/v1/users/{id}/verify",
				},
				"testing": map[string]string{
					"database": "/test/database",
					"cache":    "/test/cache",
					"config":   "/test/config",
					"responses": "/test/responses",
				},
			},
			"features": []string{
				"Dependency Container Architecture",
				"MongoDB with optimized connection pooling",
				"Redis caching with interface abstraction",
				"Structured logging with slog",
				"Standardized JSON responses",
				"Complete Users CRUD module",
				"Request validation and error handling",
				"Soft delete support",
				"Search and filtering capabilities",
				"User statistics and analytics",
				"Swagger/OpenAPI documentation",
				"Interactive API testing",
			},
		}

		response.JSONWithMessage(w, apiInfo, "Welcome to Go API Template - Phase 2 with Swagger", http.StatusOK)
	})

	// Database test endpoint (from Phase 1)
	mux.HandleFunc("GET /test/database", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Database test requested")
		
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
			"phase":       "2",
		}

		response.JSONWithMessage(w, testData, "Database test passed", http.StatusOK)
	})

	// Cache test endpoint (from Phase 1)
	mux.HandleFunc("GET /test/cache", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Cache test requested")
		
		ctx := r.Context()
		testKey := "test:cache:key:phase2"
		testValue := "Hello from Phase 2 with Swagger!"

		if err := deps.GetCache().Set(ctx, testKey, testValue, 5*time.Minute); err != nil {
			logger.Error("Failed to set cache value", err)
			response.InternalServerError(w)
			return
		}

		retrievedValue, err := deps.GetCache().Get(ctx, testKey)
		if err != nil {
			logger.Error("Failed to get cache value", err)
			response.InternalServerError(w)
			return
		}

		testData := map[string]interface{}{
			"message":         "Cache connection successful",
			"test_key":        testKey,
			"test_value":      testValue,
			"retrieved_value": retrievedValue,
			"values_match":    testValue == retrievedValue,
			"phase":           "2",
		}

		response.JSONWithMessage(w, testData, "Cache test passed", http.StatusOK)
	})

	// Configuration test endpoint (from Phase 1)
	mux.HandleFunc("GET /test/config", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Configuration test requested")
		
		config := deps.GetConfig()
		testData := map[string]interface{}{
			"message":      "Configuration loaded successfully",
			"port":         config.Port,
			"environment":  config.Environment,
			"log_level":    config.LogLevel,
			"database":     config.DatabaseName,
			"is_dev":       config.IsDevelopment(),
			"is_prod":      config.IsProduction(),
			"is_test":      config.IsTest(),
			"phase":        "2",
		}

		response.JSONWithMessage(w, testData, "Configuration test passed", http.StatusOK)
	})

	// Response formats test endpoint (from Phase 1)
	mux.HandleFunc("GET /test/responses", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Response formats test requested")
		
		format := r.URL.Query().Get("format")
		
		switch format {
		case "error":
			response.Error(w, "This is a test error from Phase 2", http.StatusBadRequest)
		case "validation":
			validationErrors := []response.ValidationError{
				response.NewValidationError("username", "Username is required", ""),
				response.NewValidationError("email", "Invalid email format", "invalid-email"),
			}
			response.ValidationErrors(w, validationErrors)
		case "not_found":
			response.NotFound(w, "Test user")
		case "unauthorized":
			response.Unauthorized(w, "Authentication required")
		case "created":
			response.Created(w, map[string]string{"id": "123", "username": "testuser"}, "")
		default:
			testData := map[string]interface{}{
				"message": "Response system working correctly - Phase 2",
				"available_formats": []string{
					"?format=error",
					"?format=validation", 
					"?format=not_found",
					"?format=unauthorized",
					"?format=created",
				},
				"phase": "2",
			}
			response.JSON(w, testData, http.StatusOK)
		}
	})

	// Root endpoint - Updated for Phase 2 with Swagger (FIX: Use /{$} for exact match)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Root endpoint accessed")
		
		welcomeData := map[string]interface{}{
			"message":     "🚀 Welcome to Go API Template - Phase 2 with Swagger",
			"version":     "1.0.0",
			"phase":       "2 - Users Module + Swagger Documentation",
			"environment": deps.GetConfig().Environment,
			"features": []string{
				"✅ Dependency Container Architecture",
				"✅ MongoDB with Connection Pooling",
				"✅ Redis Cache with Interface Abstraction",
				"✅ Structured Logging",
				"✅ Standardized JSON Responses",
				"✅ Complete Users CRUD Module",
				"✅ Request Validation",
				"✅ Search & Filtering",
				"✅ User Statistics",
				"✅ Swagger/OpenAPI Documentation",
				"✅ Interactive API Testing",
			},
			"documentation": map[string]interface{}{
				"swagger_ui":     "/swagger/",
				"description":    "Interactive API documentation and testing",
				"openapi_spec":   "/api/v1/openapi.json",
				"try_it":         "Visit /swagger/ to test the API in your browser",
			},
			"endpoints": map[string]interface{}{
				"system": map[string]string{
					"health":     "/health",
					"api_info":   "/api/v1",
					"swagger":    "/swagger/",
				},
				"users": map[string]string{
					"list_users":    "GET /api/v1/users",
					"get_user":      "GET /api/v1/users/{id}",
					"create_user":   "POST /api/v1/users",
					"update_user":   "PUT /api/v1/users/{id}",
					"delete_user":   "DELETE /api/v1/users/{id}",
					"search_users":  "GET /api/v1/users/search",
					"user_stats":    "GET /api/v1/users/stats",
					"user_profile":  "GET /api/v1/users/{id}/profile",
				},
				"testing": map[string]string{
					"db_test":       "/test/database",
					"cache_test":    "/test/cache",
					"config_test":   "/test/config",
					"response_test": "/test/responses",
				},
			},
			"next_phase": "Phase 3 - HTTP Middleware (CORS, Auth, Rate Limiting)",
		}

		response.JSONWithMessage(w, welcomeData, "Phase 2 Complete + Swagger Documentation Ready!", http.StatusOK)
	})

	logger.Info("✅ System routes configured successfully")
}