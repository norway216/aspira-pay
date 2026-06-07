// Package observability provides structured logging with OpenTelemetry context.
package observability

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context keys for trace fields.
type contextKey string

const (
	CtxTraceID   contextKey = "trace_id"
	CtxRequestID contextKey = "request_id"
	CtxPaymentID contextKey = "payment_id"
	CtxUserID    contextKey = "user_id_hash"
)

// Logger wraps zap.Logger with additional helper methods.
type Logger struct {
	*zap.Logger
	sugar *zap.SugaredLogger
}

// NewLogger creates a new production-ready logger.
func NewLogger(level string) *Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zapLevel,
	)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return &Logger{
		Logger: zapLogger,
		sugar:  zapLogger.Sugar(),
	}
}

// WithContext adds trace fields from context to the logger.
func (l *Logger) WithContext(ctx context.Context) *zap.Logger {
	fields := []zap.Field{}

	if traceID, ok := ctx.Value(CtxTraceID).(string); ok && traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if requestID, ok := ctx.Value(CtxRequestID).(string); ok && requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if paymentID, ok := ctx.Value(CtxPaymentID).(string); ok && paymentID != "" {
		fields = append(fields, zap.String("payment_id", paymentID))
	}
	if userID, ok := ctx.Value(CtxUserID).(string); ok && userID != "" {
		fields = append(fields, zap.String("user_id_hash", userID))
	}

	return l.Logger.With(fields...)
}

// Sugar returns the sugared logger for printf-style logging.
func (l *Logger) Sugar() *zap.SugaredLogger { return l.sugar }

// Sync flushes any buffered log entries.
func (l *Logger) Sync() { _ = l.Logger.Sync() }
