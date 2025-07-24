// internal/config/config.go
package config

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the application
type Config struct {
	// Server Configuration
	Port        string `envconfig:"PORT" default:"8080"`
	Environment string `envconfig:"ENV" default:"development"`
	
	// Database Configuration
	MongoURL      string `envconfig:"MONGO_URL" required:"true"`
	DatabaseName  string `envconfig:"DATABASE_NAME" default:"go_api_template"`
	
	// Redis Configuration
	RedisURL      string `envconfig:"REDIS_URL" required:"true"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB       int    `envconfig:"REDIS_DB" default:"0"`
	
	// JWT Configuration
	JWTSecret           string `envconfig:"JWT_SECRET" required:"true"`
	JWTExpirationHours  int    `envconfig:"JWT_EXPIRATION_HOURS" default:"24"`
	
	// API Configuration
	RateLimitPerMinute int `envconfig:"RATE_LIMIT_PER_MINUTE" default:"100"`
	
	// Logging Configuration
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

var instance *Config

// Load loads configuration from environment variables
// It tries to load from .env file first, then from environment
func Load() *Config {
	if instance != nil {
		return instance
	}

	// Try to load .env file (optional in production)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	instance = &Config{}
	
	// Process environment variables into config struct
	if err := envconfig.Process("", instance); err != nil {
		log.Fatalf("Failed to process environment variables: %v", err)
	}

	// Validate required configurations
	if err := instance.validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	log.Printf("Configuration loaded successfully for environment: %s", instance.Environment)
	return instance
}

// Get returns the singleton config instance
func Get() *Config {
	if instance == nil {
		return Load()
	}
	return instance
}

// validate performs basic validation on the configuration
func (c *Config) validate() error {
	// Add custom validation logic here
	if c.MongoURL == "" {
		return fmt.Errorf("MONGO_URL is required")
	}
	
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	
	// Validate JWT secret length (minimum 32 characters for security)
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	
	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return c.Environment == "test"
}

// GetServerAddress returns the complete server address
func (c *Config) GetServerAddress() string {
	return ":" + c.Port
}