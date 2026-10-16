// Package logger builds the one logger the service uses, and holds the rules
// for two things a line carries wherever it is written: the id of the request
// it belongs to, and what a panic was.
//
// The JSON encoder writes severity, message and time, which is what a log
// collector reads without being configured for us. It is the only reason this
// package differs from the platform's other services, which log ts in ISO
// 8601: a different collector reads their logs.
//
// The logger is injected into constructors and never kept in a package-level
// variable, so that a test can hand a component a logger it can read back.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a logger for the given level and format: "json" for a deployed
// service, "console" for a terminal. A level or a format it does not know is
// refused, and so is a logger that cannot be built: a service whose log goes
// nowhere, or somewhere other than it was asked to, is one nobody can see.
func New(level, format string) (*zap.Logger, error) {
	cfg, err := newConfig(level, format)
	if err != nil {
		return nil, err
	}
	built, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("logger: build: %w", err)
	}
	return built, nil
}

// newConfig assembles what New builds from. It is its own function because
// several of its settings are decisions rather than defaults, and a test reads
// them here rather than inferring them from output.
func newConfig(level, format string) (zap.Config, error) {
	parsed, err := zapcore.ParseLevel(level)
	if err != nil {
		return zap.Config{}, fmt.Errorf("logger: level: %w", err)
	}

	var cfg zap.Config
	switch format {
	case "json":
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig = cloudRunEncoderConfig()
	case "console":
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		return zap.Config{}, fmt.Errorf("logger: format %q, want json or console", format)
	}
	cfg.Level = zap.NewAtomicLevelAt(parsed)

	// The format decides how a line looks and nothing else. zap's own
	// development mode would make a terminal behave differently too — a
	// DPanic that panics, stacks from warnings on.
	cfg.Development = false

	// No stack of zap's own. It would be the stack of whoever wrote the line
	// — for a request that failed, the middleware that logs it — beside a
	// panic's own stack of the code that panicked, and on every line of an
	// error that is described by its fields.
	cfg.DisableStacktrace = true

	// No sampling. By default zap keeps the first hundred repeats of a message
	// each second and drops most of the rest, and our messages are on purpose
	// a small set of event names with the detail in the fields — so a burst of
	// refusals is exactly the shape it would throw away. Everything we count
	// is counted from these lines, and a number taken from a sample is wrong
	// precisely when something is going wrong.
	cfg.Sampling = nil

	// The log goes to stdout and the logger's own failures to stderr. A
	// collector that labels a whole stream by the descriptor it came from
	// would otherwise read every line as an error.
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	return cfg, nil
}

// cloudRunEncoderConfig renames zap's keys to the ones a log collector picks
// up: severity for the level, message for the text and time for the timestamp.
// Everything else stays as zap's production defaults.
func cloudRunEncoderConfig() zapcore.EncoderConfig {
	cfg := zap.NewProductionEncoderConfig()
	cfg.LevelKey = "severity"
	cfg.MessageKey = "message"
	cfg.TimeKey = "time"
	cfg.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	cfg.EncodeLevel = cloudRunLevelEncoder
	return cfg
}

// cloudRunLevelEncoder writes zap's levels as the severity names a collector
// understands: its scale is ERROR, WARNING, INFO, DEBUG rather than zap's
// lowercase warn and dpanic.
func cloudRunLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch level {
	case zapcore.DebugLevel:
		enc.AppendString("DEBUG")
	case zapcore.InfoLevel:
		enc.AppendString("INFO")
	case zapcore.WarnLevel:
		enc.AppendString("WARNING")
	case zapcore.ErrorLevel:
		enc.AppendString("ERROR")
	case zapcore.DPanicLevel, zapcore.PanicLevel:
		enc.AppendString("CRITICAL")
	case zapcore.FatalLevel:
		enc.AppendString("EMERGENCY")
	default:
		enc.AppendString("DEFAULT")
	}
}
