package config_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MathTrail/canvas-api/internal/config"
)

// deployed is what Cloud Run and the deployment's configuration set together:
// the name of the service, and the environment it says it is.
var deployed = []string{"K_SERVICE=canvas-api", "CANVAS_ENV=prod"}

func TestDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(nil)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v, want nil", err)
	}

	for _, field := range []struct {
		name      string
		got, want any
	}{
		{"Port", cfg.Port, config.DefaultPort},
		{"Env", cfg.Env, config.EnvLocal},
		{"LogLevel", cfg.LogLevel, config.DefaultLogLevel},
		{"LogFormat", cfg.LogFormat, config.DefaultLogFormat},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, config.DefaultReadHeaderTimeout},
		{"ReadTimeout", cfg.ReadTimeout, config.DefaultReadTimeout},
		{"WriteTimeout", cfg.WriteTimeout, config.DefaultWriteTimeout},
		{"IdleTimeout", cfg.IdleTimeout, config.DefaultIdleTimeout},
		{"ShutdownTimeout", cfg.ShutdownTimeout, config.DefaultShutdownTimeout},
	} {
		if field.got != field.want {
			t.Errorf("%s = %v, want %v", field.name, field.got, field.want)
		}
	}
	if cfg.Deployed() {
		t.Error("Deployed() = true, want false when K_SERVICE is unset")
	}
	if !cfg.Local() {
		t.Error("Local() = false, want true by default")
	}
}

func TestValuesAreRead(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom([]string{
		"PORT=9090",
		"CANVAS_ENV=prod",
		"CANVAS_LOG_LEVEL=debug",
		"CANVAS_LOG_FORMAT=console",
		"CANVAS_HTTP_READ_HEADER_TIMEOUT=2s",
		"CANVAS_HTTP_READ_TIMEOUT=3s",
		"CANVAS_HTTP_WRITE_TIMEOUT=4s",
		"CANVAS_HTTP_IDLE_TIMEOUT=5s",
		"CANVAS_SHUTDOWN_TIMEOUT=6s",
	})
	if err != nil {
		t.Fatalf("LoadFrom() error = %v, want nil", err)
	}

	for _, field := range []struct {
		name      string
		got, want any
	}{
		{"Port", cfg.Port, "9090"},
		{"Env", cfg.Env, config.EnvProd},
		{"LogLevel", cfg.LogLevel, "debug"},
		{"LogFormat", cfg.LogFormat, "console"},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 2 * time.Second},
		{"ReadTimeout", cfg.ReadTimeout, 3 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 4 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 5 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 6 * time.Second},
	} {
		if field.got != field.want {
			t.Errorf("%s = %v, want %v", field.name, field.got, field.want)
		}
	}
	if cfg.Local() {
		t.Error("Local() = true, want false for prod")
	}
}

func TestEmptyValueCountsAsUnset(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom([]string{"CANVAS_LOG_LEVEL=", "PORT=", "CANVAS_ENV="})
	if err != nil {
		t.Fatalf("LoadFrom() error = %v, want nil", err)
	}
	if cfg.LogLevel != config.DefaultLogLevel {
		t.Errorf("LogLevel = %q, want the default %q", cfg.LogLevel, config.DefaultLogLevel)
	}
	if cfg.Port != config.DefaultPort {
		t.Errorf("Port = %q, want the default %q", cfg.Port, config.DefaultPort)
	}
	if cfg.Env != config.DefaultEnv {
		t.Errorf("Env = %q, want the default %q", cfg.Env, config.DefaultEnv)
	}
}

// Nothing but the declared variables enters the reader, so a dump of it can
// never carry a credential of something else.
func TestOnlyDeclaredVariablesAreRead(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom([]string{
		"DB_PASSWORD=hunter2",
		"AWS_SECRET_ACCESS_KEY=secret",
		"PATH=/usr/bin",
		"PORT=9200",
		"port=9300",
		"canvas_env=prod",
		"k_service=canvas-api",
	})
	if err != nil {
		t.Fatalf("LoadFrom() error = %v, want nil", err)
	}
	if cfg.Port != "9200" {
		t.Errorf("Port = %q, want the declared variable, as it is written, to be read", cfg.Port)
	}
	if cfg.Env != config.DefaultEnv || cfg.Deployed() {
		t.Errorf("Env = %q, Deployed() = %v, want names in another case left unread", cfg.Env, cfg.Deployed())
	}
}

// The decoder names the variables it could not read, all of them at once.
// This is the test that notices if it ever stops.
func TestDecodeFailuresNameEveryBadVariable(t *testing.T) {
	t.Parallel()

	_, err := config.LoadFrom([]string{
		"CANVAS_HTTP_READ_TIMEOUT=half a minute",
		"CANVAS_SHUTDOWN_TIMEOUT=soon",
	})
	if !errors.Is(err, config.ErrInvalid) {
		t.Fatalf("LoadFrom() error = %v, want ErrInvalid", err)
	}
	for _, name := range []string{"CANVAS_HTTP_READ_TIMEOUT", "CANVAS_SHUTDOWN_TIMEOUT"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error = %q, want it to name %s", err, name)
		}
	}
}

// A deployment is the service Cloud Run runs, said in prod and in JSON, with a
// way out that ends before the platform's grace does.
func TestADeploymentStarts(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadFrom(append([]string{"CANVAS_SHUTDOWN_TIMEOUT=9999ms"}, deployed...))
	if err != nil {
		t.Fatalf("LoadFrom() error = %v, want nil", err)
	}
	if !cfg.Deployed() {
		t.Error("Deployed() = false, want true when K_SERVICE is set")
	}
}

// Every refusal has to name the variable behind it: a message that says only
// "invalid configuration" makes somebody read the code to deploy a service.
// What development alone may do is refused wherever K_SERVICE is set, however
// the rest of the configuration reads.
func TestRefusals(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		environ []string
		wantVar string
	}{
		{name: "the port is not a number", environ: []string{"PORT=eighty"}, wantVar: "PORT"},
		{name: "the port is zero", environ: []string{"PORT=0"}, wantVar: "PORT"},
		{name: "the port is out of range", environ: []string{"PORT=65536"}, wantVar: "PORT"},
		{name: "the port has a sign", environ: []string{"PORT=+8080"}, wantVar: "PORT"},
		{name: "the port has a leading zero", environ: []string{"PORT=08080"}, wantVar: "PORT"},
		{name: "an environment nobody knows", environ: []string{"CANVAS_ENV=dev"}, wantVar: "CANVAS_ENV"},
		{name: "a deployment that says it is local", environ: []string{"K_SERVICE=canvas-api", "CANVAS_ENV=local"}, wantVar: "CANVAS_ENV"},
		{name: "a deployment that says nothing is local by default", environ: []string{"K_SERVICE=canvas-api"}, wantVar: "CANVAS_ENV"},
		{name: "a level nobody knows", environ: []string{"CANVAS_LOG_LEVEL=loud"}, wantVar: "CANVAS_LOG_LEVEL"},
		{name: "a format nobody knows", environ: []string{"CANVAS_LOG_FORMAT=xml"}, wantVar: "CANVAS_LOG_FORMAT"},
		{name: "the console format in a deployment", environ: append([]string{"CANVAS_LOG_FORMAT=console"}, deployed...), wantVar: "CANVAS_LOG_FORMAT"},
		{name: "a read header timeout of nothing", environ: []string{"CANVAS_HTTP_READ_HEADER_TIMEOUT=0s"}, wantVar: "CANVAS_HTTP_READ_HEADER_TIMEOUT"},
		{name: "a negative read timeout", environ: []string{"CANVAS_HTTP_READ_TIMEOUT=-1s"}, wantVar: "CANVAS_HTTP_READ_TIMEOUT"},
		{name: "a write timeout of nothing", environ: []string{"CANVAS_HTTP_WRITE_TIMEOUT=0s"}, wantVar: "CANVAS_HTTP_WRITE_TIMEOUT"},
		{name: "an idle timeout of nothing", environ: []string{"CANVAS_HTTP_IDLE_TIMEOUT=0s"}, wantVar: "CANVAS_HTTP_IDLE_TIMEOUT"},
		{name: "a way out of nothing", environ: []string{"CANVAS_SHUTDOWN_TIMEOUT=0s"}, wantVar: "CANVAS_SHUTDOWN_TIMEOUT"},
		{name: "a way out too short to share", environ: []string{"CANVAS_SHUTDOWN_TIMEOUT=900ms"}, wantVar: "CANVAS_SHUTDOWN_TIMEOUT"},
		{name: "a deployed way out the platform cuts off", environ: append([]string{"CANVAS_SHUTDOWN_TIMEOUT=10s"}, deployed...), wantVar: "CANVAS_SHUTDOWN_TIMEOUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.LoadFrom(tc.environ)
			if !errors.Is(err, config.ErrInvalid) {
				t.Fatalf("LoadFrom() error = %v, want ErrInvalid", err)
			}
			if !strings.Contains(err.Error(), tc.wantVar) {
				t.Errorf("error = %q, want it to name %s", err, tc.wantVar)
			}
		})
	}
}

func TestLoadReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("PORT", "9100")
	t.Setenv("CANVAS_LOG_LEVEL", "warn")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if cfg.Port != "9100" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9100")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
}

// The drain and the close share one budget: together they are the whole way
// out, and the close keeps its quarter however long the drain could take.
func TestTheWayOutIsSharedRatherThanTakenTwice(t *testing.T) {
	t.Parallel()

	for _, budget := range []time.Duration{time.Second, config.DefaultShutdownTimeout, 7 * time.Second} {
		t.Run(budget.String(), func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{ShutdownTimeout: budget}
			if got := cfg.DrainTimeout() + cfg.CloseTimeout(); got != budget {
				t.Errorf("drain + close = %v, want the whole way out, %v", got, budget)
			}
			if cfg.CloseTimeout() != budget/4 {
				t.Errorf("close = %v, want the quarter of %v the drain leaves it", cfg.CloseTimeout(), budget)
			}
		})
	}
}
