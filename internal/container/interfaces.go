package container

import (
	"context"
	"fmt"
	"net/http"

	"go-template/internal/config"
	"go-template/internal/interfaces"

	"go.mongodb.org/mongo-driver/mongo"
)

// Dependencies container holds all application dependencies
type Dependencies struct {
	// HTTP Server components
	Mux *http.ServeMux
	
	// Configuration
	Config *config.Config
	
	// Database connections
	DB *mongo.Database
	
	// Cache connection
	Cache interfaces.CacheInterface
	
	// Logging
	Logger interfaces.LoggerInterface
	
	// Context for graceful shutdown
	Context context.Context
	Cancel  context.CancelFunc
}

// NewDependencies creates a new Dependencies container with all components initialized
func NewDependencies() *Dependencies {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Dependencies{
		Mux:     http.NewServeMux(),
		Config:  config.Load(),
		Context: ctx,
		Cancel:  cancel,
	}
}

// GetDB returns the database connection
func (d *Dependencies) GetDB() *mongo.Database {
	return d.DB
}

// GetCache returns the cache interface
func (d *Dependencies) GetCache() interfaces.CacheInterface {
	return d.Cache
}

// GetLogger returns a logger with optional component context
func (d *Dependencies) GetLogger(component string) interfaces.LoggerInterface {
	if component != "" {
		return d.Logger.With("component", component)
	}
	return d.Logger
}

// GetConfig returns the application configuration
func (d *Dependencies) GetConfig() *config.Config {
	return d.Config
}

// Close gracefully closes all connections and resources
func (d *Dependencies) Close() error {
	d.Cancel() // Cancel context to signal shutdown
	
	var errors []error
	
	// Close cache connection
	if d.Cache != nil {
		if err := d.Cache.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close cache: %w", err))
		}
	}
	
	// Close database connection
	if d.DB != nil {
		if err := d.DB.Client().Disconnect(context.Background()); err != nil {
			errors = append(errors, fmt.Errorf("failed to close database: %w", err))
		}
	}
	
	// If there were any errors, return the first one
	if len(errors) > 0 {
		return errors[0]
	}
	
	return nil
}