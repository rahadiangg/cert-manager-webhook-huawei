package huaweicloud

import (
	"context"
	"os"
	"strconv"

	"log/slog"
)

// global logger instance
var logger *slog.Logger

// LogLevel represents the logging level
type LogLevel string

const (
	// LevelDebug logs are typically voluminous, and are usually disabled in production
	LevelDebug LogLevel = "debug"
	// LevelInfo is the default logging priority
	LevelInfo LogLevel = "info"
	// LevelWarn logs are more important than Info, but don't need individual human review
	LevelWarn LogLevel = "warn"
	// LevelError logs are high-priority. If an application is running smoothly, it shouldn't generate any error-level logs
	LevelError LogLevel = "error"
)

const defaultLogLevel = LevelInfo

// InitLogger initializes the global logger with the specified log level from environment variable
func InitLogger() {
	logLevel := getLogLevel()
	var level slog.Level

	switch logLevel {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Use JSON handler for production, text handler for development
	// Can be overridden with LOG_FORMAT environment variable
	logFormat := os.Getenv("LOG_FORMAT")
	var handler slog.Handler
	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// getLogLevel retrieves the log level from environment variable or returns default
func getLogLevel() LogLevel {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		return defaultLogLevel
	}

	// Also support numeric log levels for compatibility with some systems
	if numLevel, err := strconv.Atoi(levelStr); err == nil {
		switch {
		case numLevel <= -4:
			return LevelDebug
		case numLevel == 0:
			return LevelError
		case numLevel == 4:
			return LevelWarn
		default:
			return LevelInfo
		}
	}

	return LogLevel(levelStr)
}

// Log returns the logger instance
func Log() *slog.Logger {
	if logger == nil {
		// Fallback if logger wasn't initialized
		return slog.Default()
	}
	return logger
}

// WithContext returns a logger with context
func WithContext(ctx context.Context) *slog.Logger {
	return Log().With("context", ctx)
}

// With returns a logger with additional key-value pairs
func With(args ...any) *slog.Logger {
	return Log().With(args...)
}

// Helper functions for common logging scenarios

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Log().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Log().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Log().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Log().Error(msg, args...)
}
