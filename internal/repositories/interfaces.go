// internal/repositories/interfaces.go
package repositories

import (
	"context"
	"go-template/internal/models"
)

// UserRepositoryInterface defines the contract for user data persistence
type UserRepositoryInterface interface {
	// Basic CRUD operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	SoftDelete(ctx context.Context, id string) error
	
	// List and search operations
	GetAll(ctx context.Context, params *models.UsersQueryParams) ([]*models.User, int, error)
	Search(ctx context.Context, query string, limit int) ([]*models.User, error)
	
	// Existence checks
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
	
	// Role-based queries
	GetByRole(ctx context.Context, role string, limit int) ([]*models.User, error)
	CountByRole(ctx context.Context, role string) (int, error)
	
	// Status-based queries
	GetActiveUsers(ctx context.Context, limit int) ([]*models.User, error)
	GetInactiveUsers(ctx context.Context, limit int) ([]*models.User, error)
	CountActiveUsers(ctx context.Context) (int, error)
	
	// Authentication-related
	UpdateLastLogin(ctx context.Context, id string) error
	IncrementLoginCount(ctx context.Context, id string) error
	RecordFailedLogin(ctx context.Context, id string) error
	ResetFailedLogins(ctx context.Context, id string) error
	
	// Verification and status
	MarkAsVerified(ctx context.Context, id string) error
	UpdateStatus(ctx context.Context, id string, isActive bool) error
	
	// Batch operations
	CreateMany(ctx context.Context, users []*models.User) error
	UpdateMany(ctx context.Context, filter map[string]interface{}, updates map[string]interface{}) error
	DeleteMany(ctx context.Context, ids []string) error
	
	// Statistics and analytics
	GetUserStats(ctx context.Context) (map[string]interface{}, error)
	GetUsersByDateRange(ctx context.Context, startDate, endDate string) ([]*models.User, error)
	
	// Database maintenance
	Cleanup(ctx context.Context) error // Remove soft-deleted users older than X days
}

// BaseRepositoryInterface defines common repository operations
type BaseRepositoryInterface interface {
	// Health check
	Ping(ctx context.Context) error
	
	// Collection/Table management
	EnsureIndexes(ctx context.Context) error
	DropIndexes(ctx context.Context) error
	
	// Database statistics
	GetCollectionStats(ctx context.Context) (map[string]interface{}, error)
}