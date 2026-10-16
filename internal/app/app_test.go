package app_test

import (
	"context"
	"net"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/MathTrail/canvas-api/internal/app"
	"github.com/MathTrail/canvas-api/internal/config"
)

// The server must stop because its context was cancelled — the same way a
// signal reaches it — and it must stop without an error.
func TestServerStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	container := newTestContainer(t)
	server := app.NewServer(container)
	if err := server.Listen(t.Context()); err != nil {
		t.Fatalf("Listen() error = %v, want nil", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- server.Run(ctx) }()

	// The server answers before the shutdown, so the test proves that it
	// stopped rather than that it never started.
	waitForHealthz(t, server.Addr())

	cancel()

	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5s of the context being cancelled")
	}

	resp, err := http.Get("http://" + server.Addr() + "/health") //nolint:noctx // the request is expected to fail
	if err == nil {
		_ = resp.Body.Close()
		t.Error("the server still answers after shutdown, want a refused connection")
	}
}

// Addr is read from one goroutine while Run binds the listener on another,
// which is how a caller learns the port the kernel picked. Run under the race
// detector, this is the test that says the field is guarded.
func TestAddrIsSafeWhileTheServerStarts(t *testing.T) {
	t.Parallel()

	server := app.NewServer(newTestContainer(t))

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- server.Run(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	addr := ""
	for addr == "" && time.Now().Before(deadline) {
		addr = server.Addr()
		runtime.Gosched()
	}
	if addr == "" {
		t.Fatal("Addr() stayed empty for 5s, want the bound address")
	}

	cancel()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5s of the context being cancelled")
	}
}

// Run returns only once the goroutine it serves on has finished, with every line
// of its own log written: a line written after it returned would land in a
// test that has already ended, or in a process that is already on its way out.
func TestRunLogsEverythingBeforeItReturns(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.InfoLevel)
	container := newTestContainer(t)
	container.Logger = zap.New(core)
	server := app.NewServer(container)
	if err := server.Listen(t.Context()); err != nil {
		t.Fatalf("Listen() error = %v, want nil", err)
	}

	// Cancelled before Run starts, so it turns to shutting down at once: the
	// moment a goroutine of its own is most likely still to be on its way.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := server.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	var logged []string
	for _, entry := range logs.All() {
		logged = append(logged, entry.Message)
	}
	if want := []string{"listening", "shutdown requested", "stopped"}; !slices.Equal(logged, want) {
		t.Errorf("logged by the time Run returned: %q, want %q", logged, want)
	}
}

// A request that never finishes holds the server for the drain's share of the
// way out and no longer: the close that follows has to fit in the rest before
// the platform stops the process.
func TestTheDrainLeavesTheCloseItsShare(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.ShutdownTimeout = 4 * time.Second
	container := containerFrom(t, cfg)
	core, logs := observer.New(zapcore.InfoLevel)
	container.Logger = zap.New(core)

	entered, released := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(released) })
	container.Router = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(entered)
		<-released
	})
	server := app.NewServer(container)
	if err := server.Listen(t.Context()); err != nil {
		t.Fatalf("Listen() error = %v, want nil", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- server.Run(ctx) }()
	go func() {
		resp, err := http.Get("http://" + server.Addr() + "/stuck") //nolint:noctx // cut off by the shutdown
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-entered

	started := time.Now()
	cancel()
	select {
	case err := <-stopped:
		// The stop was asked for and it happened: the request it cut off is a
		// warning, not a failure of the process.
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
		if warned := logs.FilterMessage("stopped").FilterField(zap.Bool("cut_short", true)).All(); len(warned) != 1 ||
			warned[0].Level != zapcore.WarnLevel {
			t.Errorf("stop lines = %v, want one warning that the drain was cut short", warned)
		}
	case <-time.After(2 * cfg.ShutdownTimeout):
		t.Fatalf("Run() did not return within %v of the context being cancelled", 2*cfg.ShutdownTimeout)
	}

	// Each line halfway to its neighbour, so that the drain is told apart from
	// the close's quarter below it and from the whole way out above it, a
	// second away and half a second away.
	elapsed := time.Since(started)
	if least := (cfg.CloseTimeout() + cfg.DrainTimeout()) / 2; elapsed < least {
		t.Errorf("the drain took %v, want at least %v: it waits out its share of %v",
			elapsed, least, cfg.DrainTimeout())
	}
	if most := (cfg.DrainTimeout() + cfg.ShutdownTimeout) / 2; elapsed >= most {
		t.Errorf("the drain took %v, want less than %v: it is done within its share of %v",
			elapsed, most, cfg.DrainTimeout())
	}
}

// A port that is already taken is reported when the server binds it, not
// swallowed by the goroutine that serves on it.
func TestListenReportsATakenPort(t *testing.T) {
	t.Parallel()

	first := app.NewServer(newTestContainer(t))
	if err := first.Listen(t.Context()); err != nil {
		t.Fatalf("Listen() error = %v, want nil", err)
	}
	// Nothing below reads the first server again, and the listener of a server
	// nothing can reach is closed by the finalizer of its own descriptor as
	// soon as a collection happens to run. That would hand the port back and
	// leave the second server binding a free one.
	defer runtime.KeepAlive(first)

	cfg := testConfig()
	cfg.Port = portOf(t, first.Addr())
	if err := app.NewServer(containerFrom(t, cfg)).Listen(t.Context()); err == nil {
		t.Error("Listen() error = nil, want a refusal on a port that is taken")
	}
}

// Run binds the port itself when nothing bound it first, and reports a port
// that is taken at once: the process stops at its start, rather than waiting
// for a stop that would never come.
func TestRunReportsATakenPort(t *testing.T) {
	t.Parallel()

	first := app.NewServer(newTestContainer(t))
	if err := first.Listen(t.Context()); err != nil {
		t.Fatalf("Listen() error = %v, want nil", err)
	}
	// Kept reachable to the end, so that no collection closes its listener
	// and hands the port back before the second server tries it.
	defer runtime.KeepAlive(first)

	cfg := testConfig()
	cfg.Port = portOf(t, first.Addr())
	second := app.NewServer(containerFrom(t, cfg))
	ran := make(chan error, 1)
	go func() { ran <- second.Run(t.Context()) }()
	select {
	case err := <-ran:
		if err == nil || !strings.HasPrefix(err.Error(), "app: listen on ") {
			t.Errorf("Run() error = %v, want a refusal to listen on the port that is taken", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() on a port that is taken did not return within 5s, want the refusal at once")
	}
	if addr := second.Addr(); addr != "" {
		t.Errorf("Addr() = %q after the refusal, want nothing bound", addr)
	}
}

// newTestContainer is a container built from the test configuration, closed
// when the case is over.
func newTestContainer(t *testing.T) *app.Container {
	t.Helper()

	return containerFrom(t, testConfig())
}

// containerFrom builds a container from a configuration a case has changed,
// and closes it when the case is over.
func containerFrom(t *testing.T, cfg *config.Config) *app.Container {
	t.Helper()

	container, err := app.NewContainer(t.Context(), cfg, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("NewContainer() error = %v, want nil", err)
	}
	t.Cleanup(func() { container.Close(context.Background()) })
	return container
}

func testConfig() *config.Config {
	return &config.Config{
		Port:              "0", // the kernel picks a free one
		Env:               config.EnvLocal,
		LogLevel:          "debug",
		LogFormat:         "console",
		ReadHeaderTimeout: config.DefaultReadHeaderTimeout,
		ReadTimeout:       config.DefaultReadTimeout,
		WriteTimeout:      config.DefaultWriteTimeout,
		IdleTimeout:       config.DefaultIdleTimeout,
		ShutdownTimeout:   config.DefaultShutdownTimeout,
	}
}

func portOf(t *testing.T, addr string) string {
	t.Helper()

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("address %q has no port: %v", addr, err)
	}
	return port
}

func waitForHealthz(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/health") //nolint:noctx // a probe with its own deadline loop
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no healthy answer from %s within 5s", addr)
}
