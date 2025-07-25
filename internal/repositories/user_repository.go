// internal/repositories/user_repository.go
package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-template/internal/models"
)

// UserRepository implements UserRepositoryInterface using MongoDB
type UserRepository struct {
	collection *mongo.Collection
	db         *mongo.Database
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *mongo.Database) UserRepositoryInterface {
	repo := &UserRepository{
		collection: db.Collection("users"),
		db:         db,
	}
	
	// Ensure indexes on startup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := repo.EnsureIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to ensure indexes: %v", err)
	}
	
	return repo
}

// Create inserts a new user into the database
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	// Check if username already exists
	exists, err := r.ExistsByUsername(ctx, user.Username)
	if err != nil {
		return fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return errors.New("username already exists")
	}
	
	// Check if email already exists
	exists, err = r.ExistsByEmail(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return errors.New("email already exists")
	}
	
	// Insert user
	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	
	// Update user ID with the generated one
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		user.ID = oid
	}
	
	return nil
}

// GetByID retrieves a user by their ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID format: %w", err)
	}
	
	var user models.User
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false}, // Exclude soft-deleted users
	}
	
	err = r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	
	return &user, nil
}

// GetByUsername retrieves a user by their username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	filter := bson.M{
		"username":   username,
		"deleted_at": bson.M{"$exists": false},
	}
	
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	
	return &user, nil
}

// GetByEmail retrieves a user by their email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	filter := bson.M{
		"email":      email,
		"deleted_at": bson.M{"$exists": false},
	}
	
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	
	return &user, nil
}

// Update updates a user's fields
func (r *UserRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}
	
	// Add updated_at timestamp
	updates["updated_at"] = time.Now().UTC()
	
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	update := bson.M{"$set": updates}
	
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}
	
	return nil
}

// Delete permanently deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}
	
	filter := bson.M{"_id": objectID}
	
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return errors.New("user not found")
	}
	
	return nil
}

// SoftDelete soft deletes a user by setting deleted_at timestamp
func (r *UserRepository) SoftDelete(ctx context.Context, id string) error {
	updates := map[string]interface{}{
		"deleted_at": time.Now().UTC(),
		"is_active":  false,
	}
	
	return r.Update(ctx, id, updates)
}

// GetAll retrieves users with pagination and filtering
func (r *UserRepository) GetAll(ctx context.Context, params *models.UsersQueryParams) ([]*models.User, int, error) {
	// Set defaults
	params.SetDefaults()
	
	// Build filter
	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	
	// Add search filter
	if params.Search != "" {
		filter["$or"] = []bson.M{
			{"username": bson.M{"$regex": params.Search, "$options": "i"}},
			{"email": bson.M{"$regex": params.Search, "$options": "i"}},
			{"first_name": bson.M{"$regex": params.Search, "$options": "i"}},
			{"last_name": bson.M{"$regex": params.Search, "$options": "i"}},
		}
	}
	
	// Add role filter
	if params.Role != "" {
		filter["roles"] = bson.M{"$in": []string{params.Role}}
	}
	
	// Add status filter
	if params.IsActive != nil {
		filter["is_active"] = *params.IsActive
	}
	
	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}
	
	// Build sort
	sort := bson.D{}
	sortDirection := 1
	if params.SortDir == "desc" {
		sortDirection = -1
	}
	sort = append(sort, bson.E{Key: params.SortBy, Value: sortDirection})
	
	// Build options
	opts := options.Find().
		SetSkip(int64((params.Page - 1) * params.Limit)).
		SetLimit(int64(params.Limit)).
		SetSort(sort)
	
	// Execute query
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find users: %w", err)
	}
	defer cursor.Close(ctx)
	
	// Decode results
	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, 0, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}
	
	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("cursor error: %w", err)
	}
	
	return users, int(total), nil
}

// Search performs a text search on users
func (r *UserRepository) Search(ctx context.Context, query string, limit int) ([]*models.User, error) {
	filter := bson.M{
		"deleted_at": bson.M{"$exists": false},
		"$or": []bson.M{
			{"username": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
			{"first_name": bson.M{"$regex": query, "$options": "i"}},
			{"last_name": bson.M{"$regex": query, "$options": "i"}},
		},
	}
	
	opts := options.Find().SetLimit(int64(limit))
	
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer cursor.Close(ctx)
	
	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}
	
	return users, nil
}

// ExistsByUsername checks if a username already exists
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	filter := bson.M{
		"username":   username,
		"deleted_at": bson.M{"$exists": false},
	}
	
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}
	
	return count > 0, nil
}

// ExistsByEmail checks if an email already exists
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	filter := bson.M{
		"email":      email,
		"deleted_at": bson.M{"$exists": false},
	}
	
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}
	
	return count > 0, nil
}

// ExistsByID checks if a user ID exists
func (r *UserRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, fmt.Errorf("invalid user ID format: %w", err)
	}
	
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	
	return count > 0, nil
}

// GetByRole retrieves users by role
func (r *UserRepository) GetByRole(ctx context.Context, role string, limit int) ([]*models.User, error) {
	filter := bson.M{
		"roles":      bson.M{"$in": []string{role}},
		"deleted_at": bson.M{"$exists": false},
	}
	
	opts := options.Find().SetLimit(int64(limit))
	
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by role: %w", err)
	}
	defer cursor.Close(ctx)
	
	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}
	
	return users, nil
}

// CountByRole counts users by role
func (r *UserRepository) CountByRole(ctx context.Context, role string) (int, error) {
	filter := bson.M{
		"roles":      bson.M{"$in": []string{role}},
		"deleted_at": bson.M{"$exists": false},
	}
	
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count users by role: %w", err)
	}
	
	return int(count), nil
}

// GetActiveUsers retrieves active users
func (r *UserRepository) GetActiveUsers(ctx context.Context, limit int) ([]*models.User, error) {
	filter := bson.M{
		"is_active":  true,
		"deleted_at": bson.M{"$exists": false},
	}
	
	opts := options.Find().SetLimit(int64(limit))
	
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}
	defer cursor.Close(ctx)
	
	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}
	
	return users, nil
}

// GetInactiveUsers retrieves inactive users
func (r *UserRepository) GetInactiveUsers(ctx context.Context, limit int) ([]*models.User, error) {
	filter := bson.M{
		"is_active":  false,
		"deleted_at": bson.M{"$exists": false},
	}
	
	opts := options.Find().SetLimit(int64(limit))
	
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive users: %w", err)
	}
	defer cursor.Close(ctx)
	
	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}
	
	return users, nil
}

// CountActiveUsers counts active users
func (r *UserRepository) CountActiveUsers(ctx context.Context) (int, error) {
	filter := bson.M{
		"is_active":  true,
		"deleted_at": bson.M{"$exists": false},
	}
	
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}
	
	return int(count), nil
}

// UpdateLastLogin updates user's last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	updates := map[string]interface{}{
		"last_login_at": time.Now().UTC(),
	}
	
	return r.Update(ctx, id, updates)
}

// IncrementLoginCount increments user's login count
func (r *UserRepository) IncrementLoginCount(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}
	
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	update := bson.M{
		"$inc": bson.M{"login_count": 1},
		"$set": bson.M{"updated_at": time.Now().UTC()},
	}
	
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment login count: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}
	
	return nil
}

// RecordFailedLogin records a failed login attempt
func (r *UserRepository) RecordFailedLogin(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %w", err)
	}
	
	filter := bson.M{
		"_id":        objectID,
		"deleted_at": bson.M{"$exists": false},
	}
	
	update := bson.M{
		"$inc": bson.M{"failed_logins": 1},
		"$set": bson.M{
			"last_failed_at": time.Now().UTC(),
			"updated_at":     time.Now().UTC(),
		},
	}
	
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to record failed login: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return errors.New("user not found")
	}
	
	return nil
}

// ResetFailedLogins resets failed login count
func (r *UserRepository) ResetFailedLogins(ctx context.Context, id string) error {
	updates := map[string]interface{}{
		"failed_logins":  0,
		"last_failed_at": nil,
	}
	
	return r.Update(ctx, id, updates)
}

// MarkAsVerified marks user as email verified
func (r *UserRepository) MarkAsVerified(ctx context.Context, id string) error {
	updates := map[string]interface{}{
		"is_verified":        true,
		"email_verified_at": time.Now().UTC(),
	}
	
	return r.Update(ctx, id, updates)
}

// UpdateStatus updates user's active status
func (r *UserRepository) UpdateStatus(ctx context.Context, id string, isActive bool) error {
	updates := map[string]interface{}{
		"is_active": isActive,
	}
	
	return r.Update(ctx, id, updates)
}

// CreateMany creates multiple users in a single operation
func (r *UserRepository) CreateMany(ctx context.Context, users []*models.User) error {
	if len(users) == 0 {
		return nil
	}
	
	documents := make([]interface{}, len(users))
	for i, user := range users {
		documents[i] = user
	}
	
	result, err := r.collection.InsertMany(ctx, documents)
	if err != nil {
		return fmt.Errorf("failed to create multiple users: %w", err)
	}
	
	// Update user IDs with generated ones
	for i, id := range result.InsertedIDs {
		if oid, ok := id.(primitive.ObjectID); ok && i < len(users) {
			users[i].ID = oid
		}
	}
	
	return nil
}

// UpdateMany updates multiple users matching the filter
func (r *UserRepository) UpdateMany(ctx context.Context, filter map[string]interface{}, updates map[string]interface{}) error {
	// Add updated_at timestamp
	updates["updated_at"] = time.Now().UTC()
	
	// Ensure we don't update soft-deleted users
	filter["deleted_at"] = bson.M{"$exists": false}
	
	update := bson.M{"$set": updates}
	
	_, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update multiple users: %w", err)
	}
	
	return nil
}

// DeleteMany permanently deletes multiple users
func (r *UserRepository) DeleteMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	
	objectIDs := make([]primitive.ObjectID, len(ids))
	for i, id := range ids {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return fmt.Errorf("invalid user ID format at index %d: %w", i, err)
		}
		objectIDs[i] = objectID
	}
	
	filter := bson.M{"_id": bson.M{"$in": objectIDs}}
	
	_, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete multiple users: %w", err)
	}
	
	return nil
}

// GetUserStats returns user statistics
func (r *UserRepository) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	pipeline := []bson.M{
		{"$match": bson.M{"deleted_at": bson.M{"$exists": false}}},
		{"$group": bson.M{
			"_id": nil,
			"total_users": bson.M{"$sum": 1},
			"active_users": bson.M{"$sum": bson.M{"$cond": []interface{}{
				"$is_active", 1, 0,
			}}},
			"verified_users": bson.M{"$sum": bson.M{"$cond": []interface{}{
				"$is_verified", 1, 0,
			}}},
			"avg_login_count": bson.M{"$avg": "$login_count"},
		}},
	}
	
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}
	defer cursor.Close(ctx)
	
	var result map[string]interface{}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}
	}
	
	return result, nil
}

// GetUsersByDateRange retrieves users created within a date range
func (r *UserRepository) GetUsersByDateRange(ctx context.Context, startDate, endDate string) ([]*models.User, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}
	
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %w", err)
	}
	
	// Add 24 hours to include the entire end date
	end = end.Add(24 * time.Hour)
	
	filter := bson.M{
		"created_at": bson.M{
			"$gte": start,
			"$lt":  end,
		},
		"deleted_at": bson.M{"$exists": false},
	}
	
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by date range: %w", err)
	}
	defer cursor.Close(ctx)
	
	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}
	
	return users, nil
}

// Cleanup removes soft-deleted users older than specified days
func (r *UserRepository) Cleanup(ctx context.Context) error {
	// Remove users soft-deleted more than 30 days ago
	cutoffDate := time.Now().UTC().AddDate(0, 0, -30)
	
	filter := bson.M{
		"deleted_at": bson.M{
			"$exists": true,
			"$lt":     cutoffDate,
		},
	}
	
	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to cleanup users: %w", err)
	}
	
	log.Printf("Cleaned up %d old soft-deleted users", result.DeletedCount)
	return nil
}

// Ping checks if the database connection is healthy
func (r *UserRepository) Ping(ctx context.Context) error {
	return r.db.Client().Ping(ctx, nil)
}

// EnsureIndexes creates necessary indexes for the users collection
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_users_username"),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_users_email"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_users_created_at"),
		},
		{
			Keys:    bson.D{{Key: "is_active", Value: 1}},
			Options: options.Index().SetName("idx_users_is_active"),
		},
		{
			Keys:    bson.D{{Key: "roles", Value: 1}},
			Options: options.Index().SetName("idx_users_roles"),
		},
		{
			Keys:    bson.D{{Key: "deleted_at", Value: 1}},
			Options: options.Index().SetName("idx_users_deleted_at"),
		},
	}
	
	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	
	return nil
}

// DropIndexes removes all custom indexes
func (r *UserRepository) DropIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().DropAll(ctx)
	return err
}

// GetCollectionStats returns collection statistics
func (r *UserRepository) GetCollectionStats(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := r.db.RunCommand(ctx, bson.M{
		"collStats": "users",
	}).Decode(&result)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}
	
	return result, nil
}