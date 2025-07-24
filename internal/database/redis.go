package database

import (
	"context"
	"encoding/json"
	"fmt"
	"go-template/internal/interfaces"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements the CacheInterface using Redis
type RedisCache struct {
	client redis.UniversalClient
}

// ConnectRedis establishes a connection to Redis and returns a CacheInterface implementation
func ConnectRedis(redisURL, password string, db int) (interfaces.CacheInterface, error) {
	log.Printf("Connecting to Redis at %s...", redisURL)

	// Configure Redis client options for optimal performance
	options := &redis.Options{
		Addr:     redisURL,
		Password: password,
		DB:       db,
		
		// Connection pool settings
		PoolSize:     100,                // Maximum number of socket connections
		MinIdleConns: 10,                 // Minimum number of idle connections
		PoolTimeout:  30 * time.Second,   // Amount of time client waits for connection
		
		// Timeouts
		DialTimeout:  5 * time.Second,  // Timeout for socket connection
		ReadTimeout:  3 * time.Second,  // Timeout for socket reads
		WriteTimeout: 3 * time.Second,  // Timeout for socket writes
		
		// Retry settings
		MaxRetries:      3,                    // Maximum number of retries before giving up
		MinRetryBackoff: 8 * time.Millisecond,  // Minimum backoff between each retry
		MaxRetryBackoff: 512 * time.Millisecond, // Maximum backoff between each retry
	}

	// Create Redis client
	client := redis.NewClient(options)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis")

	// Wrap in our CacheInterface implementation
	cache := &RedisCache{client: client}
	
	// Start periodic stats logging
	go cache.logStats()

	return cache, nil
}

// Get retrieves a value from cache
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return result, err
}

// Set stores a value in cache with expiration
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// Serialize value to JSON if it's not a string
	var serialized interface{}
	switch v := value.(type) {
	case string:
		serialized = v
	case []byte:
		serialized = v
	default:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to serialize value for key %s: %w", key, err)
		}
		serialized = jsonBytes
	}

	return r.client.Set(ctx, key, serialized, expiration).Err()
}

// Delete removes one or more keys from cache
func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists in cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

// MGet retrieves multiple values at once
func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return []interface{}{}, nil
	}
	return r.client.MGet(ctx, keys...).Result()
}

// MSet sets multiple key-value pairs at once
func (r *RedisCache) MSet(ctx context.Context, pairs ...interface{}) error {
	if len(pairs) == 0 {
		return nil
	}
	return r.client.MSet(ctx, pairs...).Err()
}

// Increment increments a numeric value
func (r *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// Expire sets expiration time for a key
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the time to live for a key
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// FlushAll removes all keys from the database
func (r *RedisCache) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}

// Ping checks if Redis connection is healthy
func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	log.Println("Closing Redis connection...")
	err := r.client.Close()
	if err == nil {
		log.Println("Redis connection closed successfully")
	}
	return err
}

// Publish publishes a message to a channel
func (r *RedisCache) Publish(ctx context.Context, channel string, message interface{}) error {
	// Serialize message to JSON
	var payload interface{}
	switch v := message.(type) {
	case string:
		payload = v
	case []byte:
		payload = v
	default:
		jsonBytes, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to serialize message for channel %s: %w", channel, err)
		}
		payload = jsonBytes
	}

	return r.client.Publish(ctx, channel, payload).Err()
}

// Subscribe subscribes to one or more channels
func (r *RedisCache) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return r.client.Subscribe(ctx, channels...)
}

// logStats logs Redis connection statistics periodically
func (r *RedisCache) logStats() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		// Get Redis info
		info, err := r.client.Info(ctx, "stats").Result()
		if err == nil {
			log.Printf("Redis Stats: %s", info)
		}
		
		// Get pool stats
		stats := r.client.PoolStats()
		log.Printf("Redis Pool Stats - Hits: %d, Misses: %d, Timeouts: %d, TotalConns: %d, IdleConns: %d",
			stats.Hits, stats.Misses, stats.Timeouts, stats.TotalConns, stats.IdleConns)
		
		cancel()
	}
}

// Helper functions for common cache patterns

// GetJSON retrieves and unmarshals a JSON value from cache
func (r *RedisCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	
	return json.Unmarshal([]byte(data), dest)
}

// SetJSON marshals and stores a value as JSON in cache
func (r *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Set(ctx, key, value, expiration)
}

// Remember implements the cache-aside pattern
// It tries to get from cache first, if not found, calls the fetcher function and caches the result
func (r *RedisCache) Remember(ctx context.Context, key string, expiration time.Duration, fetcher func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if data, err := r.Get(ctx, key); err == nil {
		return data, nil
	}
	
	// Not in cache, call fetcher
	value, err := fetcher()
	if err != nil {
		return nil, err
	}
	
	// Store in cache (fire and forget)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r.Set(bgCtx, key, value, expiration)
	}()
	
	return value, nil
}