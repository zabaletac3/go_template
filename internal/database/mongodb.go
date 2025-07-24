package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// ConnectMongoDB establishes a connection to MongoDB with optimized settings
func ConnectMongoDB(mongoURL, databaseName string) (*mongo.Database, error) {
	// Configure client options for optimal performance
	clientOptions := options.Client().
		ApplyURI(mongoURL).
		// Connection pool settings
		SetMaxPoolSize(100).                // Maximum number of connections in the pool
		SetMinPoolSize(10).                 // Minimum number of connections to maintain
		SetMaxConnIdleTime(30 * time.Second). // Close connections after 30s of inactivity
		// Timeout settings
		SetConnectTimeout(30 * time.Second).     // Timeout for initial connection
		SetServerSelectionTimeout(30 * time.Second). // Timeout for server selection
		SetSocketTimeout(30 * time.Second).      // Timeout for socket operations
		// Retry settings
		SetRetryWrites(true).  // Enable retryable writes
		SetRetryReads(true).   // Enable retryable reads
		// Monitoring
		SetHeartbeatInterval(10 * time.Second). // Health check interval
		SetLocalThreshold(15 * time.Millisecond) // Local threshold for server selection

	// Create context with timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("Connecting to MongoDB at %s...", mongoURL)

	// Create MongoDB client
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Ping MongoDB to verify connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	log.Printf("Successfully connected to MongoDB database: %s", databaseName)

	// Return the database instance
	database := client.Database(databaseName)
	
	// Log database stats for monitoring
	go logDatabaseStats(database)

	return database, nil
}

// logDatabaseStats logs database connection statistics periodically
func logDatabaseStats(db *mongo.Database) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		// Get database stats
		var result map[string]interface{}
		err := db.RunCommand(ctx, map[string]interface{}{"dbStats": 1}).Decode(&result)
		
		if err == nil {
			log.Printf("MongoDB Stats - Collections: %v, Objects: %v, Data Size: %v KB", 
				result["collections"], 
				result["objects"], 
				result["dataSize"])
		}
		
		cancel()
	}
}

// PingMongoDB checks if MongoDB connection is healthy
func PingMongoDB(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Client().Ping(ctx, readpref.Primary())
}

// CloseMongoDB gracefully closes the MongoDB connection
func CloseMongoDB(db *mongo.Database) error {
	if db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Closing MongoDB connection...")
	
	if err := db.Client().Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
	}

	log.Println("MongoDB connection closed successfully")
	return nil
}

// CreateIndexes creates commonly used indexes for better performance
// This will be expanded in Phase 2 when we add specific collections
func CreateIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("Creating database indexes...")

	// Example: Create index for users collection (will be used in Phase 2)
	usersCollection := db.Collection("users")
	
	// Index for email field (unique)
	emailIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"email": 1},
		Options: options.Index().SetUnique(true).SetName("idx_users_email"),
	}
	
	// Index for username field (unique)
	usernameIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"username": 1},
		Options: options.Index().SetUnique(true).SetName("idx_users_username"),
	}
	
	// Index for created_at field (for sorting)
	createdAtIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"created_at": -1},
		Options: options.Index().SetName("idx_users_created_at"),
	}

	// Create indexes
	_, err := usersCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		emailIndex,
		usernameIndex,
		createdAtIndex,
	})

	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database indexes created successfully")
	return nil
}

// GetCollectionNames returns all collection names in the database
func GetCollectionNames(db *mongo.Database) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	names, err := db.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collection names: %w", err)
	}

	return names, nil
}