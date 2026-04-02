package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a zap.Logger configured for the given log level and output
// format. Supported formats: "console" (colored, dev-friendly) and "json"
// (structured, production). Unknown levels fall back to info.
func NewLogger(serviceName, level, format string) *zap.Logger {
	var zapLevel zapcore.Level
	if zapLevel.UnmarshalText([]byte(level)) != nil {
		zapLevel = zapcore.InfoLevel
	}

	var cfg zap.Config
	if format == "console" {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "ts"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	cfg.InitialFields = map[string]any{"service": serviceName}
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := cfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build zap logger: %v\n", err)
		return zap.NewNop()
	}
	return logger
}
