package container

import (
	"context"
	"fmt"
	"go-template/internal/database"
	"go-template/internal/interfaces"
	"log"
	"log/slog"
	"os"
)

// Initialize sets up all dependencies and returns a fully configured Dependencies container
func (d *Dependencies) Initialize() error {
	log.Println("Initializing application dependencies...")

	// Initialize logger first (needed by other components)
	if err := d.initLogger(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger := d.GetLogger("container")
	logger.Info("Logger initialized successfully")

	// Initialize database connection
	if err := d.initDatabase(); err != nil {
		logger.Error("Failed to initialize database", err)
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.Info("Database initialized successfully")

	// Initialize cache connection
	if err := d.initCache(); err != nil {
		logger.Error("Failed to initialize cache", err)
		return fmt.Errorf("failed to initialize cache: %w", err)
	}
	logger.Info("Cache initialized successfully")

	logger.Info("All dependencies initialized successfully")
	return nil
}

// initLogger initializes the structured logger
func (d *Dependencies) initLogger() error {
	// Configure log level based on config
	var logLevel slog.Level
	switch d.Config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: d.Config.IsDevelopment(),
	}

	// Use JSON handler for production, text handler for development
	var handler slog.Handler
	if d.Config.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Create logger and wrap it in our LoggerInterface implementation
	logger := slog.New(handler)
	d.Logger = &StructuredLogger{logger: logger}

	return nil
}

// initDatabase initializes the MongoDB connection
func (d *Dependencies) initDatabase() error {
	db, err := database.ConnectMongoDB(d.Config.MongoURL, d.Config.DatabaseName)
	if err != nil {
		return err
	}

	d.DB = db
	return nil
}

// initCache initializes the Redis cache connection
func (d *Dependencies) initCache() error {
	cache, err := database.ConnectRedis(
		d.Config.RedisURL,
		d.Config.RedisPassword,
		d.Config.RedisDB,
	)
	if err != nil {
		return err
	}

	d.Cache = cache
	return nil
}

// StructuredLogger implements interfaces.LoggerInterface using slog
type StructuredLogger struct {
	logger *slog.Logger
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, err error, args ...interface{}) {
	if err != nil {
		args = append([]interface{}{"error", err.Error()}, args...)
	}
	l.logger.Error(msg, args...)
}

// With returns a new logger with additional context
func (l *StructuredLogger) With(args ...interface{}) interfaces.LoggerInterface {
	return &StructuredLogger{
		logger: l.logger.With(args...),
	}
}

// WithContext returns a new logger with context
func (l *StructuredLogger) WithContext(ctx context.Context) interfaces.LoggerInterface {
	return &StructuredLogger{
		logger: l.logger.With("request_id", getRequestIDFromContext(ctx)),
	}
}

// Log logs at the specified level
func (l *StructuredLogger) Log(ctx context.Context, level slog.Level, msg string, args ...interface{}) {
	l.logger.Log(ctx, level, msg, args...)
}

// getRequestIDFromContext extracts request ID from context
func getRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return ""
}