// Package config reads the service's configuration from the environment.
//
// Environment variables are the only source: there is no file, no flag and no
// remote source. They are read here and nowhere else, so that every value has
// one origin and one validation. Viper does the reading — as it does in the
// platform's other services — which keeps the names, the defaults and the
// types of a growing table in one declaration.
//
// Only the variables the service actually uses are here. A variable arrives
// with the code that consumes it, because a required variable with no reader
// is a deployment that fails for no reason.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// The two environments the service knows. Local is a developer's machine,
// where the conveniences of development may be switched on; prod is the one
// deployment, where none of them may.
const (
	EnvLocal = "local"
	EnvProd  = "prod"
)

// Defaults for every optional variable. They are named constants rather than
// literals in LoadFrom so that the documentation and the code cannot drift.
const (
	DefaultPort              = "8080"
	DefaultEnv               = EnvLocal
	DefaultLogLevel          = "info"
	DefaultLogFormat         = "json"
	DefaultReadHeaderTimeout = 5 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 60 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
	DefaultShutdownTimeout   = 9 * time.Second
)

// MinShutdownTimeout is the shortest way out a service may be given: the close
// after the drain has a quarter of it, and a quarter of less is no time to
// close anything in.
const MinShutdownTimeout = time.Second

// CloudRunStopGrace is how long Cloud Run waits after it asks an instance to
// stop before it kills it. A deployed way out has to end before then, or the
// close it keeps time for never runs.
const CloudRunStopGrace = 10 * time.Second

// ErrInvalid is returned by LoadFrom and Validate; callers branch on it with
// errors.Is.
var ErrInvalid = errors.New("config: invalid")

// Config is the whole configuration of the process. Every field names the
// variable it comes from, so that a refusal and the deployment that caused it
// use the same word.
type Config struct {
	// Port is the TCP port to listen on.
	Port string `mapstructure:"PORT"`
	// Env is the environment the service runs in: local or prod.
	Env string `mapstructure:"CANVAS_ENV"`
	// LogLevel is a zap level name: debug, info, warn or error.
	LogLevel string `mapstructure:"CANVAS_LOG_LEVEL"`
	// LogFormat is "json" for a log collector and "console" to read by eye.
	LogFormat string `mapstructure:"CANVAS_LOG_FORMAT"`

	// ReadHeaderTimeout is how long a client may take to send its headers.
	ReadHeaderTimeout time.Duration `mapstructure:"CANVAS_HTTP_READ_HEADER_TIMEOUT"`
	// ReadTimeout is how long a client may take to send its whole request.
	ReadTimeout time.Duration `mapstructure:"CANVAS_HTTP_READ_TIMEOUT"`
	// WriteTimeout is how long the server may take to send its answer.
	WriteTimeout time.Duration `mapstructure:"CANVAS_HTTP_WRITE_TIMEOUT"`
	// IdleTimeout is how long a connection is kept open between requests.
	IdleTimeout time.Duration `mapstructure:"CANVAS_HTTP_IDLE_TIMEOUT"`
	// ShutdownTimeout is the whole way out once a stop is asked for: the
	// server's drain and the close after it together. The platform stops the
	// process ten seconds after asking it to, and time spent past that is not
	// spent at all, so the default stops a second short of it: the signal takes
	// a moment to arrive, and the last lines are written after the close.
	ShutdownTimeout time.Duration `mapstructure:"CANVAS_SHUTDOWN_TIMEOUT"`

	// Service is the name K_SERVICE carries. Cloud Run sets it on every
	// service and a developer's machine does not, which is what makes it the
	// gate for everything meant for development only: no value of ours can
	// open that gate in a deployment.
	Service string `mapstructure:"K_SERVICE"`
}

// Deployed reports whether the process is running as a deployed service.
func (c *Config) Deployed() bool { return c.Service != "" }

// Local reports whether the service runs on a developer's machine, where the
// conveniences of development may be switched on.
func (c *Config) Local() bool { return c.Env == EnvLocal }

// DrainTimeout is the part of the way out the server waits for the requests
// still in flight: three quarters of it.
//
// The drain and the close after it share one budget rather than take one
// each, in fixed shares, so that a slow request cannot eat the time the close
// needs.
func (c *Config) DrainTimeout() time.Duration { return c.ShutdownTimeout - c.CloseTimeout() }

// CloseTimeout is the part of the way out kept for closing what the service ran
// on, once the server has drained or a start has failed halfway: a quarter of
// it.
func (c *Config) CloseTimeout() time.Duration { return c.ShutdownTimeout / 4 }

// Load reads the process environment.
func Load() (*Config, error) { return LoadFrom(os.Environ()) }

// LoadFrom reads the configuration from a set of KEY=VALUE strings, in the
// shape os.Environ returns them, applies the defaults and validates the
// result. The source is an argument so that a test can hand over an
// environment instead of changing the one the process runs in.
//
// A variable set to an empty string counts as unset: an empty value is what a
// deployment leaves behind when it stops setting something, and it should not
// silently override a default with nothing.
func LoadFrom(environ []string) (*Config, error) {
	v := viper.New()

	// Every key is declared, including the ones whose default is empty, so
	// that the names, the defaults and the types sit in one place.
	v.SetDefault("PORT", DefaultPort)
	v.SetDefault("CANVAS_ENV", DefaultEnv)
	v.SetDefault("CANVAS_LOG_LEVEL", DefaultLogLevel)
	v.SetDefault("CANVAS_LOG_FORMAT", DefaultLogFormat)
	v.SetDefault("CANVAS_HTTP_READ_HEADER_TIMEOUT", DefaultReadHeaderTimeout)
	v.SetDefault("CANVAS_HTTP_READ_TIMEOUT", DefaultReadTimeout)
	v.SetDefault("CANVAS_HTTP_WRITE_TIMEOUT", DefaultWriteTimeout)
	v.SetDefault("CANVAS_HTTP_IDLE_TIMEOUT", DefaultIdleTimeout)
	v.SetDefault("CANVAS_SHUTDOWN_TIMEOUT", DefaultShutdownTimeout)
	v.SetDefault("K_SERVICE", "")

	// Only the declared keys are taken from the environment. Everything else a
	// process carries — credentials of other systems, tokens, the shell's own
	// variables — has no business inside this reader, where a future dump of
	// its contents would carry them straight into a log. A name is matched as
	// it is written: the environment tells PORT from port, and so does this.
	// Viper keeps its keys in lower case, so they are matched in upper case.
	declared := make(map[string]struct{}, len(v.AllKeys()))
	for _, key := range v.AllKeys() {
		declared[strings.ToUpper(key)] = struct{}{}
	}
	for _, entry := range environ {
		name, value, found := strings.Cut(entry, "=")
		if !found || value == "" {
			continue
		}
		if _, ours := declared[name]; !ours {
			continue
		}
		v.Set(name, value)
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		// The decoder names every variable it could not read, and it names all
		// of them rather than the first, so its message is passed through as
		// it is. A test holds it to that.
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate reports the first problem it finds, naming the variable behind it.
func (c *Config) Validate() error {
	for _, check := range []func() error{
		c.validatePort,
		c.validateEnv,
		c.validateLogging,
		c.validateTimeouts,
	} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

// validatePort refuses a port the server could not listen on, and one written
// in any other way than plain digits: a sign or a leading zero would be read
// as the same port and logged as something else.
func (c *Config) validatePort() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 || strconv.Itoa(port) != c.Port {
		return fmt.Errorf("%w: PORT must be a number from 1 to 65535, got %q", ErrInvalid, c.Port)
	}
	return nil
}

// validateEnv refuses an environment the service does not know, and the local
// one in a deployment. Local is what switches development's conveniences on,
// so a deployment that says it is local is refused outright rather than
// warned about: whatever its variables say, a service on Cloud Run is prod.
func (c *Config) validateEnv() error {
	switch c.Env {
	case EnvLocal, EnvProd:
	default:
		return fmt.Errorf("%w: CANVAS_ENV must be %s or %s, got %q", ErrInvalid, EnvLocal, EnvProd, c.Env)
	}
	if c.Local() && c.Deployed() {
		return fmt.Errorf("%w: CANVAS_ENV must be %s when K_SERVICE is set", ErrInvalid, EnvProd)
	}
	return nil
}

// validateLogging refuses a level or a format the logger does not know, and
// the console format in a deployment.
func (c *Config) validateLogging() error {
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("%w: CANVAS_LOG_LEVEL must be debug, info, warn or error, got %q", ErrInvalid, c.LogLevel)
	}

	switch c.LogFormat {
	case "json", "console":
	default:
		return fmt.Errorf("%w: CANVAS_LOG_FORMAT must be json or console, got %q", ErrInvalid, c.LogFormat)
	}
	// The console format is for a person at a terminal. It escapes nothing, so a
	// newline a request carries would start a line of its own, and a collector
	// reads no severity out of it: a deployment writes JSON.
	if c.LogFormat == "console" && c.Deployed() {
		return fmt.Errorf("%w: CANVAS_LOG_FORMAT must be json when K_SERVICE is set", ErrInvalid)
	}
	return nil
}

// validateTimeouts refuses a duration nothing could run under, naming its
// variable: every one has to be positive, and the way out long enough to be
// shared.
func (c *Config) validateTimeouts() error {
	for _, timeout := range []struct {
		name  string
		value time.Duration
	}{
		{"CANVAS_HTTP_READ_HEADER_TIMEOUT", c.ReadHeaderTimeout},
		{"CANVAS_HTTP_READ_TIMEOUT", c.ReadTimeout},
		{"CANVAS_HTTP_WRITE_TIMEOUT", c.WriteTimeout},
		{"CANVAS_HTTP_IDLE_TIMEOUT", c.IdleTimeout},
		{"CANVAS_SHUTDOWN_TIMEOUT", c.ShutdownTimeout},
	} {
		if timeout.value <= 0 {
			return fmt.Errorf("%w: %s must be a positive duration, got %v", ErrInvalid, timeout.name, timeout.value)
		}
	}
	if c.ShutdownTimeout < MinShutdownTimeout {
		return fmt.Errorf("%w: CANVAS_SHUTDOWN_TIMEOUT must be at least %v, got %v",
			ErrInvalid, MinShutdownTimeout, c.ShutdownTimeout)
	}
	if c.Deployed() && c.ShutdownTimeout >= CloudRunStopGrace {
		return fmt.Errorf("%w: CANVAS_SHUTDOWN_TIMEOUT must be under %v when K_SERVICE is set, got %v",
			ErrInvalid, CloudRunStopGrace, c.ShutdownTimeout)
	}
	return nil
}
