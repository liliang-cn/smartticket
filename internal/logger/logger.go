package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/company/smartticket/internal/config"
)

// Logger wraps the zap logger with additional functionality
type Logger struct {
	*zap.Logger
	config *config.LoggerConfig
}

// NewLogger creates a new structured logger instance
func NewLogger(cfg *config.LoggerConfig) (*Logger, error) {
	// Build zap configuration
	zapConfig := zap.NewProductionConfig()

	// Set log level
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level '%s': %w", cfg.Level, err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Set output encoder based on format
	if cfg.Format == "text" {
		zapConfig.Encoding = "console"
		zapConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig.Encoding = "json"
		zapConfig.EncoderConfig = zap.NewProductionEncoderConfig()
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.MessageKey = "message"
		zapConfig.EncoderConfig.LevelKey = "level"
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.CallerKey = "caller"
	}

	// Set output destination
	var output io.Writer
	switch cfg.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "file":
		if cfg.FilePath == "" {
			return nil, fmt.Errorf("file path is required when output is 'file'")
		}
		// Create rotating file writer
		output = &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   true,
		}
	default:
		return nil, fmt.Errorf("unsupported output type: %s", cfg.Output)
	}

	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}

	// Build logger
	zapLogger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}

	// If using file output, redirect
	if cfg.Output == "file" {
		zapLogger = zapLogger.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core {
			return zapcore.NewCore(
				zapcore.NewJSONEncoder(zapConfig.EncoderConfig),
				zapcore.AddSync(output),
				level,
			)
		}))
	}

	logger := &Logger{
		Logger: zapLogger,
		config: cfg,
	}

	return logger, nil
}

// parseLogLevel converts string log level to zapcore.Level
func parseLogLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unsupported log level: %s", level)
	}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(requestID string) *zap.Logger {
	return l.Logger.With(zap.String("request_id", requestID))
}

// WithTenantID adds tenant ID to logger context
func (l *Logger) WithTenantID(tenantID uint) *zap.Logger {
	return l.Logger.With(zap.Uint("tenant_id", tenantID))
}

// WithUserID adds user ID to logger context
func (l *Logger) WithUserID(userID uint) *zap.Logger {
	return l.Logger.With(zap.Uint("user_id", userID))
}

// WithFields adds multiple fields to logger context
func (l *Logger) WithFields(fields map[string]interface{}) *zap.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		switch v := value.(type) {
		case string:
			zapFields = append(zapFields, zap.String(key, v))
		case int:
			zapFields = append(zapFields, zap.Int(key, v))
		case uint:
			zapFields = append(zapFields, zap.Uint(key, v))
		case int64:
			zapFields = append(zapFields, zap.Int64(key, v))
		case float64:
			zapFields = append(zapFields, zap.Float64(key, v))
		case bool:
			zapFields = append(zapFields, zap.Bool(key, v))
		case time.Time:
			zapFields = append(zapFields, zap.Time(key, v))
		case time.Duration:
			zapFields = append(zapFields, zap.Duration(key, v))
		case error:
			zapFields = append(zapFields, zap.Error(v))
		default:
			zapFields = append(zapFields, zap.Any(key, v))
		}
	}
	return l.Logger.With(zapFields...)
}

// LogRequest logs HTTP request information
func (l *Logger) LogRequest(method, path, remoteAddr, userAgent string, statusCode int, duration time.Duration, requestID string) {
	l.Logger.Info("HTTP Request",
		zap.String("method", method),
		zap.String("path", path),
		zap.String("remote_addr", remoteAddr),
		zap.String("user_agent", userAgent),
		zap.Int("status_code", statusCode),
		zap.Duration("duration", duration),
		zap.String("request_id", requestID),
	)
}

// LogError logs error with context
func (l *Logger) LogError(err error, message string, fields ...zap.Field) {
	if len(fields) > 0 {
		l.Logger.Error(message, append(fields, zap.Error(err))...)
	} else {
		l.Logger.Error(message, zap.Error(err))
	}
}

// LogDatabaseOperation logs database operations
func (l *Logger) LogDatabaseOperation(operation, table string, duration time.Duration, rowsAffected int64, err error) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("table", table),
		zap.Duration("duration", duration),
		zap.Int64("rows_affected", rowsAffected),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Logger.Error("Database operation failed", fields...)
	} else {
		l.Logger.Debug("Database operation completed", fields...)
	}
}

// LogSecurityEvent logs security-related events
func (l *Logger) LogSecurityEvent(event, userID, ipAddress, userAgent string, success bool) {
	l.Logger.Info("Security event",
		zap.String("event", event),
		zap.String("user_id", userID),
		zap.String("ip_address", ipAddress),
		zap.String("user_agent", userAgent),
		zap.Bool("success", success),
	)
}

// LogBusinessEvent logs business-related events
func (l *Logger) LogBusinessEvent(event string, tenantID, userID uint, details map[string]interface{}) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.Uint("tenant_id", tenantID),
		zap.Uint("user_id", userID),
	}

	for key, value := range details {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case uint:
			fields = append(fields, zap.Uint(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}

	l.Logger.Info("Business event", fields...)
}

// LogPerformanceMetric logs performance metrics
func (l *Logger) LogPerformanceMetric(metric string, value float64, unit string, tags map[string]string) {
	fields := []zap.Field{
		zap.String("metric", metric),
		zap.Float64("value", value),
		zap.String("unit", unit),
	}

	for key, tag := range tags {
		fields = append(fields, zap.String(fmt.Sprintf("tag_%s", key), tag))
	}

	l.Logger.Info("Performance metric", fields...)
}

// GetConfig returns the logger configuration
func (l *Logger) GetConfig() *config.LoggerConfig {
	return l.config
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.Logger.Core().Enabled(zapcore.DebugLevel)
}

// IsInfoEnabled returns true if info logging is enabled
func (l *Logger) IsInfoEnabled() bool {
	return l.Logger.Core().Enabled(zapcore.InfoLevel)
}

// IsWarnEnabled returns true if warning logging is enabled
func (l *Logger) IsWarnEnabled() bool {
	return l.Logger.Core().Enabled(zapcore.WarnLevel)
}

// IsErrorEnabled returns true if error logging is enabled
func (l *Logger) IsErrorEnabled() bool {
	return l.Logger.Core().Enabled(zapcore.ErrorLevel)
}

// Global logger instance
var globalLogger *Logger

// InitializeGlobalLogger initializes the global logger instance
func InitializeGlobalLogger(cfg *config.LoggerConfig) error {
	logger, err := NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize global logger: %w", err)
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Fallback to a default logger if global logger is not initialized
		logger, _ := NewLogger(&config.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		})
		globalLogger = logger
	}
	return globalLogger
}

// Global convenience functions that use the global logger instance

func Debug(msg string, fields ...zap.Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	GetGlobalLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	GetGlobalLogger().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	GetGlobalLogger().Panic(msg, fields...)
}

func Sync() error {
	return GetGlobalLogger().Sync()
}

func WithRequestID(requestID string) *zap.Logger {
	return GetGlobalLogger().WithRequestID(requestID)
}

func WithTenantID(tenantID uint) *zap.Logger {
	return GetGlobalLogger().WithTenantID(tenantID)
}

func WithUserID(userID uint) *zap.Logger {
	return GetGlobalLogger().WithUserID(userID)
}

func WithFields(fields map[string]interface{}) *zap.Logger {
	return GetGlobalLogger().WithFields(fields)
}

func LogRequest(method, path, remoteAddr, userAgent string, statusCode int, duration time.Duration, requestID string) {
	GetGlobalLogger().LogRequest(method, path, remoteAddr, userAgent, statusCode, duration, requestID)
}

func LogError(err error, message string, fields ...zap.Field) {
	GetGlobalLogger().LogError(err, message, fields...)
}

func LogDatabaseOperation(operation, table string, duration time.Duration, rowsAffected int64, err error) {
	GetGlobalLogger().LogDatabaseOperation(operation, table, duration, rowsAffected, err)
}

func LogSecurityEvent(event, userID, ipAddress, userAgent string, success bool) {
	GetGlobalLogger().LogSecurityEvent(event, userID, ipAddress, userAgent, success)
}

func LogBusinessEvent(event string, tenantID, userID uint, details map[string]interface{}) {
	GetGlobalLogger().LogBusinessEvent(event, tenantID, userID, details)
}

func LogPerformanceMetric(metric string, value float64, unit string, tags map[string]string) {
	GetGlobalLogger().LogPerformanceMetric(metric, value, unit, tags)
}
