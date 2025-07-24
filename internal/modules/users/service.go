// internal/modules/users/service.go
package users

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-template/internal/interfaces"
	"go-template/internal/models"
	"go-template/internal/repositories"
)

// UserService handles business logic for user operations
type UserService struct {
	repo   repositories.UserRepositoryInterface
	cache  interfaces.CacheInterface
	logger interfaces.LoggerInterface
}

// Cache key constants
const (
	CacheKeyUser         = "user:id:%s"
	CacheKeyUserByEmail  = "user:email:%s"
	CacheKeyUserUsername = "user:username:%s"
	CacheKeyUserStats    = "user:stats"
	CacheKeyUserList     = "user:list:%s" // Hash of query params
	CacheKeyUserExists   = "user:exists:%s:%s" // type:value (email:user@example.com)
	
	// Cache expiration times
	UserCacheExpiration      = 15 * time.Minute
	UserListCacheExpiration  = 5 * time.Minute
	UserStatsCacheExpiration = 30 * time.Minute
	UserExistsCacheExpiration = 10 * time.Minute
)

// NewUserService creates a new UserService instance
func NewUserService(
	repo repositories.UserRepositoryInterface,
	cache interfaces.CacheInterface,
	logger interfaces.LoggerInterface,
) *UserService {
	return &UserService{
		repo:   repo,
		cache:  cache,
		logger: logger.With("service", "users"),
	}
}

// CreateUser creates a new user with validation and cache management
func (s *UserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	s.logger.Info("Creating new user", "username", req.Username, "email", req.Email)
	
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		s.logger.Warn("User creation validation failed", "errors", errors)
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}
	
	// Check if username or email already exists (with cache)
	exists, err := s.checkUserExists(ctx, "username", req.Username)
	if err != nil {
		s.logger.Error("Failed to check username existence", err)
		return nil, fmt.Errorf("failed to validate username: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("username '%s' already exists", req.Username)
	}
	
	exists, err = s.checkUserExists(ctx, "email", req.Email)
	if err != nil {
		s.logger.Error("Failed to check email existence", err)
		return nil, fmt.Errorf("failed to validate email: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email '%s' already exists", req.Email)
	}
	
	// Create user model
	user, err := models.NewUser(req.Username, req.Email, req.Password)
	if err != nil {
		s.logger.Error("Failed to create user model", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Set optional fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	
	// Save to database
	if err := s.repo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to save user to database", err)
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	
	// Cache the new user
	s.cacheUser(ctx, user)
	
	// Invalidate related caches
	s.invalidateUserListCaches(ctx)
	s.invalidateUserStats(ctx)
	
	s.logger.Info("User created successfully", "user_id", user.GetIDString(), "username", user.Username)
	return user, nil
}

// GetUserByID retrieves a user by ID with caching
func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	s.logger.Debug("Getting user by ID", "user_id", id)
	
	// Try cache first
	cacheKey := fmt.Sprintf(CacheKeyUser, id)
	if cached, err := s.getUserFromCache(ctx, cacheKey); err == nil && cached != nil {
		s.logger.Debug("User found in cache", "user_id", id)
		return cached, nil
	}
	
	// Get from database
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get user from database", err, "user_id", id)
		return nil, err
	}
	
	// Cache the user
	s.cacheUser(ctx, user)
	
	s.logger.Debug("User retrieved from database and cached", "user_id", id)
	return user, nil
}

// GetUserByEmail retrieves a user by email with caching
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	s.logger.Debug("Getting user by email", "email", email)
	
	// Try cache first
	cacheKey := fmt.Sprintf(CacheKeyUserByEmail, email)
	if cached, err := s.getUserFromCache(ctx, cacheKey); err == nil && cached != nil {
		s.logger.Debug("User found in cache", "email", email)
		return cached, nil
	}
	
	// Get from database
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to get user by email", err, "email", email)
		return nil, err
	}
	
	// Cache the user with multiple keys
	s.cacheUser(ctx, user)
	
	s.logger.Debug("User retrieved from database and cached", "email", email)
	return user, nil
}

// GetUserByUsername retrieves a user by username with caching
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	s.logger.Debug("Getting user by username", "username", username)
	
	// Try cache first
	cacheKey := fmt.Sprintf(CacheKeyUserUsername, username)
	if cached, err := s.getUserFromCache(ctx, cacheKey); err == nil && cached != nil {
		s.logger.Debug("User found in cache", "username", username)
		return cached, nil
	}
	
	// Get from database
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		s.logger.Error("Failed to get user by username", err, "username", username)
		return nil, err
	}
	
	// Cache the user
	s.cacheUser(ctx, user)
	
	s.logger.Debug("User retrieved from database and cached", "username", username)
	return user, nil
}

// UpdateUser updates a user with validation and cache management
func (s *UserService) UpdateUser(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	s.logger.Info("Updating user", "user_id", id)
	
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		s.logger.Warn("User update validation failed", "errors", errors)
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}
	
	// Get current user
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Check for unique constraint violations
	updates := req.ToMap()
	
	if newUsername, ok := updates["username"].(string); ok && newUsername != user.Username {
		exists, err := s.checkUserExists(ctx, "username", newUsername)
		if err != nil {
			return nil, fmt.Errorf("failed to validate username: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("username '%s' already exists", newUsername)
		}
	}
	
	if newEmail, ok := updates["email"].(string); ok && newEmail != user.Email {
		exists, err := s.checkUserExists(ctx, "email", newEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to validate email: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("email '%s' already exists", newEmail)
		}
	}
	
	// Update in database
	if err := s.repo.Update(ctx, id, updates); err != nil {
		s.logger.Error("Failed to update user in database", err, "user_id", id)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	// Invalidate caches
	s.invalidateUserCaches(ctx, user)
	s.invalidateUserListCaches(ctx)
	
	// Get updated user
	updatedUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get updated user", err, "user_id", id)
		return nil, fmt.Errorf("failed to retrieve updated user: %w", err)
	}
	
	// Cache updated user
	s.cacheUser(ctx, updatedUser)
	
	s.logger.Info("User updated successfully", "user_id", id)
	return updatedUser, nil
}

// DeleteUser soft deletes a user and manages cache
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	s.logger.Info("Deleting user", "user_id", id)
	
	// Get user for cache invalidation
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	
	// Soft delete in database
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		s.logger.Error("Failed to delete user", err, "user_id", id)
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	// Invalidate caches
	s.invalidateUserCaches(ctx, user)
	s.invalidateUserListCaches(ctx)
	s.invalidateUserStats(ctx)
	
	s.logger.Info("User deleted successfully", "user_id", id)
	return nil
}

// GetUsers retrieves users with pagination and caching
func (s *UserService) GetUsers(ctx context.Context, params *models.UsersQueryParams) ([]*models.User, int, error) {
	s.logger.Debug("Getting users list", "page", params.Page, "limit", params.Limit)
	
	// Set defaults
	params.SetDefaults()
	
	// Try cache first (only for default queries without search/filters)
	if s.isCacheableQuery(params) {
		cacheKey := s.buildUserListCacheKey(params)
		if cached, err := s.getUserListFromCache(ctx, cacheKey); err == nil && cached != nil {
			s.logger.Debug("User list found in cache")
			// Convert UserResponse to User models
			users := make([]*models.User, len(cached.Users))
			for i, userResp := range cached.Users {
				user := &models.User{}
				userJSON, _ := json.Marshal(userResp)
				json.Unmarshal(userJSON, user)
				users[i] = user
			}
			return users, cached.Total, nil
		}
	}
	
	// Get from database
	users, total, err := s.repo.GetAll(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get users from database", err)
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}
	
	// Cache result if cacheable
	if s.isCacheableQuery(params) {
		cacheKey := s.buildUserListCacheKey(params)
		result := &models.UserListResponse{
			Users: make([]models.UserResponse, len(users)),
			Total: total,
			Page:  params.Page,
			Limit: params.Limit,
		}
		
		for i, user := range users {
			result.Users[i] = user.ToUserResponse()
		}
		
		s.cacheUserList(ctx, cacheKey, result)
	}
	
	s.logger.Debug("Users retrieved from database", "count", len(users), "total", total)
	return users, total, nil
}

// SearchUsers performs search on users
func (s *UserService) SearchUsers(ctx context.Context, query string, limit int) ([]*models.User, error) {
	s.logger.Debug("Searching users", "query", query, "limit", limit)
	
	if query == "" {
		return []*models.User{}, nil
	}
	
	users, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		s.logger.Error("Failed to search users", err, "query", query)
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	
	s.logger.Debug("User search completed", "query", query, "count", len(users))
	return users, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, id string, req *models.ChangePasswordRequest) error {
	s.logger.Info("Changing user password", "user_id", id)
	
	// Validate request
	if errors := req.Validate(); len(errors) > 0 {
		s.logger.Warn("Password change validation failed", "errors", errors)
		return fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}
	
	// Get user
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	
	// Verify current password
	if !user.CheckPassword(req.CurrentPassword) {
		s.logger.Warn("Invalid current password provided", "user_id", id)
		return fmt.Errorf("current password is incorrect")
	}
	
	// Set new password
	if err := user.SetPassword(req.NewPassword); err != nil {
		s.logger.Error("Failed to set new password", err, "user_id", id)
		return fmt.Errorf("failed to set new password: %w", err)
	}
	
	// Update in database
	updates := map[string]interface{}{
		"password": user.Password,
		"salt":     user.Salt,
	}
	
	if err := s.repo.Update(ctx, id, updates); err != nil {
		s.logger.Error("Failed to update password in database", err, "user_id", id)
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	// Invalidate user caches
	s.invalidateUserCaches(ctx, user)
	
	s.logger.Info("Password changed successfully", "user_id", id)
	return nil
}

// VerifyUser marks a user as verified
func (s *UserService) VerifyUser(ctx context.Context, id string) error {
	s.logger.Info("Verifying user", "user_id", id)
	
	// Get user
	user, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	
	if user.IsVerified {
		return fmt.Errorf("user is already verified")
	}
	
	// Mark as verified in database
	if err := s.repo.MarkAsVerified(ctx, id); err != nil {
		s.logger.Error("Failed to verify user", err, "user_id", id)
		return fmt.Errorf("failed to verify user: %w", err)
	}
	
	// Invalidate caches
	s.invalidateUserCaches(ctx, user)
	s.invalidateUserStats(ctx)
	
	s.logger.Info("User verified successfully", "user_id", id)
	return nil
}

// GetUserStats returns user statistics with caching
func (s *UserService) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	s.logger.Debug("Getting user statistics")
	
	// Try cache first
	cacheKey := CacheKeyUserStats
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var stats map[string]interface{}
		if json.Unmarshal([]byte(cached), &stats) == nil {
			s.logger.Debug("User stats found in cache")
			return stats, nil
		}
	}
	
	// Get from database
	stats, err := s.repo.GetUserStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get user stats", err)
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	
	// Cache the stats
	if statsJSON, err := json.Marshal(stats); err == nil {
		s.cache.Set(ctx, cacheKey, statsJSON, UserStatsCacheExpiration)
	}
	
	s.logger.Debug("User stats retrieved from database and cached")
	return stats, nil
}

// Helper methods for caching

// getUserFromCache retrieves a user from cache
func (s *UserService) getUserFromCache(ctx context.Context, key string) (*models.User, error) {
	cached, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	
	var user models.User
	if err := json.Unmarshal([]byte(cached), &user); err != nil {
		return nil, err
	}
	
	return &user, nil
}

// cacheUser stores a user in cache with multiple keys
func (s *UserService) cacheUser(ctx context.Context, user *models.User) {
	userJSON, err := json.Marshal(user)
	if err != nil {
		s.logger.Error("Failed to marshal user for caching", err)
		return
	}
	
	// Cache with multiple keys for different access patterns
	keys := []string{
		fmt.Sprintf(CacheKeyUser, user.GetIDString()),
		fmt.Sprintf(CacheKeyUserByEmail, user.Email),
		fmt.Sprintf(CacheKeyUserUsername, user.Username),
	}
	
	for _, key := range keys {
		if err := s.cache.Set(ctx, key, userJSON, UserCacheExpiration); err != nil {
			s.logger.Error("Failed to cache user", err, "cache_key", key)
		}
	}
}

// invalidateUserCaches removes user from all cache keys
func (s *UserService) invalidateUserCaches(ctx context.Context, user *models.User) {
	keys := []string{
		fmt.Sprintf(CacheKeyUser, user.GetIDString()),
		fmt.Sprintf(CacheKeyUserByEmail, user.Email),
		fmt.Sprintf(CacheKeyUserUsername, user.Username),
		fmt.Sprintf(CacheKeyUserExists, "email", user.Email),
		fmt.Sprintf(CacheKeyUserExists, "username", user.Username),
	}
	
	for _, key := range keys {
		if err := s.cache.Delete(ctx, key); err != nil {
			s.logger.Error("Failed to invalidate cache", err, "cache_key", key)
		}
	}
}

// invalidateUserListCaches removes user list caches
func (s *UserService) invalidateUserListCaches(ctx context.Context) {
	// In a real implementation, you might use cache tagging or patterns
	// For now, we'll use a simple approach
	pattern := "user:list:*"
	s.logger.Debug("Invalidating user list caches", "pattern", pattern)
	// Note: This is a simplified approach. In production, consider using cache tagging
}

// invalidateUserStats removes user stats cache
func (s *UserService) invalidateUserStats(ctx context.Context) {
	if err := s.cache.Delete(ctx, CacheKeyUserStats); err != nil {
		s.logger.Error("Failed to invalidate user stats cache", err)
	}
}

// checkUserExists checks if a user exists by field with caching
func (s *UserService) checkUserExists(ctx context.Context, field, value string) (bool, error) {
	cacheKey := fmt.Sprintf(CacheKeyUserExists, field, value)
	
	// Try cache first
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		return cached == "true", nil
	}
	
	// Check database
	var exists bool
	var err error
	
	switch field {
	case "email":
		exists, err = s.repo.ExistsByEmail(ctx, value)
	case "username":
		exists, err = s.repo.ExistsByUsername(ctx, value)
	default:
		return false, fmt.Errorf("unsupported field: %s", field)
	}
	
	if err != nil {
		return false, err
	}
	
	// Cache the result
	cacheValue := "false"
	if exists {
		cacheValue = "true"
	}
	s.cache.Set(ctx, cacheKey, cacheValue, UserExistsCacheExpiration)
	
	return exists, nil
}

// getUserListFromCache retrieves user list from cache
func (s *UserService) getUserListFromCache(ctx context.Context, key string) (*models.UserListResponse, error) {
	cached, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	
	var result models.UserListResponse
	if err := json.Unmarshal([]byte(cached), &result); err != nil {
		return nil, err
	}
	
	// Convert back to User models
	users := make([]*models.User, len(result.Users))
	for i, userResp := range result.Users {
		// This is a simplified conversion - in production you might want a more robust approach
		user := &models.User{}
		userJSON, _ := json.Marshal(userResp)
		json.Unmarshal(userJSON, user)
		users[i] = user
	}
	
	return &models.UserListResponse{
		Users: result.Users,
		Total: result.Total,
		Page:  result.Page,
		Limit: result.Limit,
	}, nil
}

// cacheUserList stores user list in cache
func (s *UserService) cacheUserList(ctx context.Context, key string, list *models.UserListResponse) {
	listJSON, err := json.Marshal(list)
	if err != nil {
		s.logger.Error("Failed to marshal user list for caching", err)
		return
	}
	
	if err := s.cache.Set(ctx, key, listJSON, UserListCacheExpiration); err != nil {
		s.logger.Error("Failed to cache user list", err)
	}
}

// isCacheableQuery determines if a query can be cached
func (s *UserService) isCacheableQuery(params *models.UsersQueryParams) bool {
	// Only cache simple queries without search or complex filters
	return params.Search == "" && params.Role == "" && params.IsActive == nil
}

// buildUserListCacheKey creates a cache key for user list queries
func (s *UserService) buildUserListCacheKey(params *models.UsersQueryParams) string {
	return fmt.Sprintf(CacheKeyUserList, fmt.Sprintf("page:%d:limit:%d:sort:%s:%s", 
		params.Page, params.Limit, params.SortBy, params.SortDir))
}