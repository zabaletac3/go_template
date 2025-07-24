package interfaces

import (
	"context"
	"log/slog"
)

// LoggerInterface defines the contract for structured logging
type LoggerInterface interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, err error, args ...interface{})
	With(args ...interface{}) LoggerInterface
	WithContext(ctx context.Context) LoggerInterface
	Log(ctx context.Context, level slog.Level, msg string, args ...interface{})
} 