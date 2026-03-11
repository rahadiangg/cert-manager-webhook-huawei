package huaweicloud

import (
	"context"
	"os"
	"strconv"
	"time"

	"log/slog"
)

// global logger instance
var logger *slog.Logger

// slog type aliases for convenience
type (
	// Attr is an alias for slog.Attr for convenience
	Attr = slog.Attr
)

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
// Deprecated: Use context-aware logging functions (InfoCtx, DebugCtx, etc.) instead
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

// Context-aware logging functions
// These use slog's built-in context support for proper context propagation

// DebugCtx logs a debug message with context
func DebugCtx(ctx context.Context, msg string, args ...any) {
	Log().DebugContext(ctx, msg, args...)
}

// InfoCtx logs an info message with context
func InfoCtx(ctx context.Context, msg string, args ...any) {
	Log().InfoContext(ctx, msg, args...)
}

// WarnCtx logs a warning message with context
func WarnCtx(ctx context.Context, msg string, args ...any) {
	Log().WarnContext(ctx, msg, args...)
}

// ErrorCtx logs an error message with context
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	Log().ErrorContext(ctx, msg, args...)
}

// LogAttrs functions for more efficient logging with pre-constructed attributes
// These avoid the allocation overhead of key-value pairs

// DebugAttrs logs a debug message with slog.Attr values
func DebugAttrs(msg string, args ...Attr) {
	Log().LogAttrs(context.TODO(), slog.LevelDebug, msg, args...)
}

// InfoAttrs logs an info message with slog.Attr values
func InfoAttrs(msg string, args ...Attr) {
	Log().LogAttrs(context.TODO(), slog.LevelInfo, msg, args...)
}

// WarnAttrs logs a warning message with slog.Attr values
func WarnAttrs(msg string, args ...Attr) {
	Log().LogAttrs(context.TODO(), slog.LevelWarn, msg, args...)
}

// ErrorAttrs logs an error message with slog.Attr values
func ErrorAttrs(msg string, args ...Attr) {
	Log().LogAttrs(context.TODO(), slog.LevelError, msg, args...)
}

// DebugAttrsCtx logs a debug message with context and slog.Attr values
func DebugAttrsCtx(ctx context.Context, msg string, args ...Attr) {
	Log().LogAttrs(ctx, slog.LevelDebug, msg, args...)
}

// InfoAttrsCtx logs an info message with context and slog.Attr values
func InfoAttrsCtx(ctx context.Context, msg string, args ...Attr) {
	Log().LogAttrs(ctx, slog.LevelInfo, msg, args...)
}

// WarnAttrsCtx logs a warning message with context and slog.Attr values
func WarnAttrsCtx(ctx context.Context, msg string, args ...Attr) {
	Log().LogAttrs(ctx, slog.LevelWarn, msg, args...)
}

// ErrorAttrsCtx logs an error message with context and slog.Attr values
func ErrorAttrsCtx(ctx context.Context, msg string, args ...Attr) {
	Log().LogAttrs(ctx, slog.LevelError, msg, args...)
}

// Common attribute constructors
// These provide slog-style attribute constructors for common types

// Any returns an Attr for any value
func Any(key string, value any) Attr {
	return slog.Any(key, value)
}

// String returns an Attr for a string value
func String(key, value string) Attr {
	return slog.String(key, value)
}

// Int64 returns an Attr for an int64 value
func Int64(key string, value int64) Attr {
	return slog.Int64(key, value)
}

// Int returns an Attr for an int value
func Int(key string, value int) Attr {
	return slog.Int(key, value)
}

// Uint64 returns an Attr for a uint64 value
func Uint64(key string, value uint64) Attr {
	return slog.Uint64(key, value)
}

// Float64 returns an Attr for a float64 value
func Float64(key string, value float64) Attr {
	return slog.Float64(key, value)
}

// Bool returns an Attr for a bool value
func Bool(key string, value bool) Attr {
	return slog.Bool(key, value)
}

// Time returns an Attr for a time.Time value
func Time(key string, value time.Time) Attr {
	return slog.Time(key, value)
}

// Duration returns an Attr for a time.Duration value
func Duration(key string, value time.Duration) Attr {
	return slog.Duration(key, value)
}

// Group returns an Attr for a group of attributes
func Group(key string, args ...Attr) Attr {
	// Convert []Attr to []any for slog.Group
	anyArgs := make([]any, len(args))
	for i, arg := range args {
		anyArgs[i] = arg
	}
	return slog.Group(key, anyArgs...)
}

// Err returns an Attr for an error value
// Use this for consistent error logging: Error("operation failed", Err(err))
func Err(err error) Attr {
	return slog.Any("error", err)
}

// ErrKey returns an Attr for an error with a custom key
func ErrKey(key string, err error) Attr {
	return slog.Any(key, err)
}
