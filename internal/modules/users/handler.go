package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go-template/internal/interfaces"
	"go-template/internal/models"
	"go-template/internal/shared/response"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	service *UserService
	logger  interfaces.LoggerInterface
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(service *UserService, logger interfaces.LoggerInterface) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger.With("handler", "users"),
	}
}

// GetUsers handles GET /api/v1/users
// @Summary Get all users
// @Description Get all users with pagination and filtering options
// @Tags Users
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Param search query string false "Search in username, email, first_name, last_name"
// @Param role query string false "Filter by role" Enums(user, admin, moderator)
// @Param is_active query bool false "Filter by active status"
// @Param sort_by query string false "Sort field" default(created_at) Enums(created_at, updated_at, username, email, first_name, last_name, login_count)
// @Param sort_dir query string false "Sort direction" default(desc) Enums(asc, desc)
// @Success 200 {object} response.Response{data=models.UserListResponse,meta=response.Meta} "List of users with pagination metadata"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Invalid query parameters"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Getting users list")
	
	// Parse query parameters
	params, err := h.parseUsersQueryParams(r)
	if err != nil {
		h.logger.Warn("Invalid query parameters", "error", err.Error())
		response.BadRequest(w, err.Error())
		return
	}
	
	// Get users from service
	users, total, err := h.service.GetUsers(r.Context(), params)
	if err != nil {
		h.logger.Error("Failed to get users", err)
		response.InternalServerError(w)
		return
	}
	
	// Convert to response DTOs
	userResponses := make([]models.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToUserResponse()
	}
	
	// Create response with metadata
	userList := models.UserListResponse{
		Users: userResponses,
		Total: total,
		Page:  params.Page,
		Limit: params.Limit,
	}
	
	// Create pagination metadata
	meta := response.NewMeta(params.Page, params.Limit, total)
	
	response.JSONWithMeta(w, userList, meta, http.StatusOK)
	h.logger.Info("Users retrieved successfully", "count", len(users), "total", total)
}

// GetUser handles GET /api/v1/users/{id}
// @Summary Get user by ID
// @Description Get a specific user by their unique identifier
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(objectid) example(507f1f77bcf86cd799439011)
// @Success 200 {object} response.Response{data=models.UserResponse} "User information"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Invalid user ID format"
// @Failure 404 {object} response.Response{error=response.ErrorInfo} "User not found"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "User ID is required")
		return
	}
	
	h.logger.Info("Getting user", "user_id", id)
	
	// Get user from service
	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.logger.Warn("User not found", "user_id", id)
			response.NotFound(w, "User")
			return
		}
		h.logger.Error("Failed to get user", err, "user_id", id)
		response.InternalServerError(w)
		return
	}
	
	// Convert to response DTO
	userResponse := user.ToUserResponse()
	
	response.JSON(w, userResponse, http.StatusOK)
	h.logger.Info("User retrieved successfully", "user_id", id)
}

// CreateUser handles POST /api/v1/users
// @Summary Create a new user
// @Description Create a new user account with validation
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.CreateUserRequest true "User creation data"
// @Success 201 {object} response.Response{data=models.UserResponse} "User created successfully"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Validation error or invalid request body"
// @Failure 409 {object} response.Response{error=response.ErrorInfo} "Username or email already exists"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Creating new user")
	
	// Parse request body
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err.Error())
		response.BadRequest(w, "Invalid request body format")
		return
	}
	
	// Create user through service
	user, err := h.service.CreateUser(r.Context(), &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			h.logger.Warn("User creation conflict", "error", err.Error())
			response.ErrorWithCode(w, "CONFLICT", err.Error(), http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "validation failed") {
			h.logger.Warn("User creation validation failed", "error", err.Error())
			response.BadRequest(w, err.Error())
			return
		}
		h.logger.Error("Failed to create user", err)
		response.InternalServerError(w)
		return
	}
	
	// Convert to response DTO
	userResponse := user.ToUserResponse()
	
	response.Created(w, userResponse, "User created successfully")
	h.logger.Info("User created successfully", "user_id", user.GetIDString(), "username", user.Username)
}

// UpdateUser handles PATCH /api/v1/users/{id}
// @Summary Update user
// @Description Partially update user information with validation (only provided fields are updated)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(objectid) example(507f1f77bcf86cd799439011)
// @Param user body models.UpdateUserRequest true "User update data (partial)"
// @Success 200 {object} response.Response{data=models.UserResponse} "User updated successfully"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Validation error or invalid request body"
// @Failure 404 {object} response.Response{error=response.ErrorInfo} "User not found"
// @Failure 409 {object} response.Response{error=response.ErrorInfo} "Username or email already exists"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/{id} [patch]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "User ID is required")
		return
	}
	
	h.logger.Info("Updating user", "user_id", id)
	
	// Parse request body
	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err.Error())
		response.BadRequest(w, "Invalid request body format")
		return
	}
	
	// Update user through service
	user, err := h.service.UpdateUser(r.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.logger.Warn("User not found for update", "user_id", id)
			response.NotFound(w, "User")
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			h.logger.Warn("User update conflict", "error", err.Error())
			response.ErrorWithCode(w, "CONFLICT", err.Error(), http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "validation failed") {
			h.logger.Warn("User update validation failed", "error", err.Error())
			response.BadRequest(w, err.Error())
			return
		}
		h.logger.Error("Failed to update user", err, "user_id", id)
		response.InternalServerError(w)
		return
	}
	
	// Convert to response DTO
	userResponse := user.ToUserResponse()
	
	response.Updated(w, userResponse, "User updated successfully")
	h.logger.Info("User updated successfully", "user_id", id)
}

// DeleteUser handles DELETE /api/v1/users/{id}
// @Summary Delete user
// @Description Soft delete a user account (user data is preserved but marked as deleted)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(objectid) example(507f1f77bcf86cd799439011)
// @Success 200 {object} response.Response "User deleted successfully"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Invalid user ID format"
// @Failure 404 {object} response.Response{error=response.ErrorInfo} "User not found"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "User ID is required")
		return
	}
	
	h.logger.Info("Deleting user", "user_id", id)
	
	// Delete user through service
	err := h.service.DeleteUser(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.logger.Warn("User not found for deletion", "user_id", id)
			response.NotFound(w, "User")
			return
		}
		h.logger.Error("Failed to delete user", err, "user_id", id)
		response.InternalServerError(w)
		return
	}
	
	response.Deleted(w, "User deleted successfully")
	h.logger.Info("User deleted successfully", "user_id", id)
}

// SearchUsers handles GET /api/v1/users/search
// @Summary Search users
// @Description Search users by username, email, first name, or last name
// @Tags Users
// @Accept json
// @Produce json
// @Param q query string true "Search query" minlength(1) maxlength(100) example(john)
// @Param limit query int false "Maximum results" default(10) minimum(1) maximum(50)
// @Success 200 {object} response.Response{data=[]models.UserProfileResponse} "List of matching user profiles"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Missing or invalid search query"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/search [get]
func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	// Get search query
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		response.BadRequest(w, "Search query is required")
		return
	}
	
	// Get limit parameter
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 50 {
			limit = parsedLimit
		}
	}
	
	h.logger.Info("Searching users", "query", query, "limit", limit)
	
	// Search users through service
	users, err := h.service.SearchUsers(r.Context(), query, limit)
	if err != nil {
		h.logger.Error("Failed to search users", err, "query", query)
		response.InternalServerError(w)
		return
	}
	
	// Convert to public profile responses (limited information)
	userProfiles := make([]models.UserProfileResponse, len(users))
	for i, user := range users {
		userProfiles[i] = user.ToUserProfileResponse()
	}
	
	response.JSON(w, userProfiles, http.StatusOK)
	h.logger.Info("User search completed", "query", query, "count", len(users))
}

// ChangePassword handles PATCH /api/v1/users/{id}/password
// @Summary Change user password
// @Description Change a user's password with current password verification
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(objectid) example(507f1f77bcf86cd799439011)
// @Param password body models.ChangePasswordRequest true "Password change data"
// @Success 200 {object} response.Response "Password changed successfully"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Validation error or incorrect current password"
// @Failure 404 {object} response.Response{error=response.ErrorInfo} "User not found"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/{id}/password [patch]
func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "User ID is required")
		return
	}
	
	h.logger.Info("Changing user password", "user_id", id)
	
	// Parse request body
	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err.Error())
		response.BadRequest(w, "Invalid request body format")
		return
	}
	
	// Change password through service
	err := h.service.ChangePassword(r.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, "User")
			return
		}
		if strings.Contains(err.Error(), "validation failed") || strings.Contains(err.Error(), "incorrect") {
			h.logger.Warn("Password change validation failed", "error", err.Error())
			response.BadRequest(w, err.Error())
			return
		}
		h.logger.Error("Failed to change password", err, "user_id", id)
		response.InternalServerError(w)
		return
	}
	
	response.JSONWithMessage(w, nil, "Password changed successfully", http.StatusOK)
	h.logger.Info("Password changed successfully", "user_id", id)
}

// VerifyUser handles PATCH /api/v1/users/{id}/verify
// @Summary Verify user email
// @Description Mark a user's email as verified
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(objectid) example(507f1f77bcf86cd799439011)
// @Success 200 {object} response.Response "User verified successfully"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "User already verified or invalid ID"
// @Failure 404 {object} response.Response{error=response.ErrorInfo} "User not found"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/{id}/verify [patch]
func (h *UserHandler) VerifyUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "User ID is required")
		return
	}
	
	h.logger.Info("Verifying user", "user_id", id)
	
	// Verify user through service
	err := h.service.VerifyUser(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, "User")
			return
		}
		if strings.Contains(err.Error(), "already verified") {
			response.BadRequest(w, err.Error())
			return
		}
		h.logger.Error("Failed to verify user", err, "user_id", id)
		response.InternalServerError(w)
		return
	}
	
	response.JSONWithMessage(w, nil, "User verified successfully", http.StatusOK)
	h.logger.Info("User verified successfully", "user_id", id)
}

// GetUserStats handles GET /api/v1/users/stats
// @Summary Get user statistics
// @Description Get aggregated user statistics including total users, active users, verified users, etc.
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=object} "User statistics"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/stats [get]
func (h *UserHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Getting user statistics")
	
	// Get stats from service
	stats, err := h.service.GetUserStats(r.Context())
	if err != nil {
		h.logger.Error("Failed to get user stats", err)
		response.InternalServerError(w)
		return
	}
	
	response.JSON(w, stats, http.StatusOK)
	h.logger.Info("User statistics retrieved successfully")
}

// GetUserProfile handles GET /api/v1/users/{id}/profile
// @Summary Get user public profile
// @Description Get a user's public profile information (limited data for privacy)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(objectid) example(507f1f77bcf86cd799439011)
// @Success 200 {object} response.Response{data=models.UserProfileResponse} "User public profile"
// @Failure 400 {object} response.Response{error=response.ErrorInfo} "Invalid user ID format"
// @Failure 404 {object} response.Response{error=response.ErrorInfo} "User not found"
// @Failure 500 {object} response.Response{error=response.ErrorInfo} "Internal server error"
// @Router /api/v1/users/{id}/profile [get]
func (h *UserHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := r.PathValue("id")
	if id == "" {
		response.BadRequest(w, "User ID is required")
		return
	}
	
	h.logger.Info("Getting user profile", "user_id", id)
	
	// Get user from service
	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			response.NotFound(w, "User")
			return
		}
		h.logger.Error("Failed to get user profile", err, "user_id", id)
		response.InternalServerError(w)
		return
	}
	
	// Convert to public profile response
	profile := user.ToUserProfileResponse()
	
	response.JSON(w, profile, http.StatusOK)
	h.logger.Info("User profile retrieved successfully", "user_id", id)
}

// Helper methods

// parseUsersQueryParams parses and validates query parameters for user listing
func (h *UserHandler) parseUsersQueryParams(r *http.Request) (*models.UsersQueryParams, error) {
	params := &models.UsersQueryParams{}
	
	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		} else {
			return nil, fmt.Errorf("invalid page parameter")
		}
	}
	
	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		} else {
			return nil, fmt.Errorf("invalid limit parameter (must be between 1 and 100)")
		}
	}
	
	// Parse search
	params.Search = strings.TrimSpace(r.URL.Query().Get("search"))
	
	// Parse role
	params.Role = strings.TrimSpace(r.URL.Query().Get("role"))
	
	// Parse is_active
	if activeStr := r.URL.Query().Get("is_active"); activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			params.IsActive = &active
		} else {
			return nil, fmt.Errorf("invalid is_active parameter (must be true or false)")
		}
	}
	
	// Parse sort_by
	params.SortBy = strings.TrimSpace(r.URL.Query().Get("sort_by"))
	if params.SortBy != "" {
		// Validate allowed sort fields
		allowedSortFields := []string{"created_at", "updated_at", "username", "email", "first_name", "last_name", "login_count"}
		validSort := false
		for _, field := range allowedSortFields {
			if params.SortBy == field {
				validSort = true
				break
			}
		}
		if !validSort {
			return nil, fmt.Errorf("invalid sort_by parameter (allowed: %v)", allowedSortFields)
		}
	}
	
	// Parse sort_dir
	params.SortDir = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_dir")))
	if params.SortDir != "" && params.SortDir != "asc" && params.SortDir != "desc" {
		return nil, fmt.Errorf("invalid sort_dir parameter (must be 'asc' or 'desc')")
	}
	
	// Set defaults
	params.SetDefaults()
	
	return params, nil
}