// internal/modules/users/routes.go
package users

import (
	"go-template/internal/container"
	"go-template/internal/repositories"
)

// RegisterRoutes registers all user-related routes
// This function is completely self-contained and handles its own dependency injection
func RegisterRoutes(deps *container.Dependencies) {
	logger := deps.GetLogger("users")
	logger.Info("Registering user module routes")

	// Internal dependency injection for the users module
	repo := repositories.NewUserRepository(deps.GetDB())
	service := NewUserService(repo, deps.GetCache(), logger)
	handler := NewUserHandler(service, logger)

	// Get the HTTP multiplexer
	mux := deps.Mux

	// User CRUD endpoints
	mux.HandleFunc("GET /api/v1/users", handler.GetUsers)
	mux.HandleFunc("GET /api/v1/users/{id}", handler.GetUser)
	mux.HandleFunc("POST /api/v1/users", handler.CreateUser)
	mux.HandleFunc("PATCH /api/v1/users/{id}", handler.UpdateUser)  
	mux.HandleFunc("DELETE /api/v1/users/{id}", handler.DeleteUser)

	// User search endpoint
	mux.HandleFunc("GET /api/v1/users/search", handler.SearchUsers)

	// User statistics endpoint
	mux.HandleFunc("GET /api/v1/users/stats", handler.GetUserStats)

	// User profile endpoints
	mux.HandleFunc("GET /api/v1/users/{id}/profile", handler.GetUserProfile)

	// User account management endpoints
	mux.HandleFunc("PATCH /api/v1/users/{id}/password", handler.ChangePassword)
	mux.HandleFunc("PATCH /api/v1/users/{id}/verify", handler.VerifyUser)

	logger.Info("✅ User module routes registered successfully", 
		"endpoints", 9, 
		"base_path", "/api/v1/users")
}