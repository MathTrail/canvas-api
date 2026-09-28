package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The service answers its health probe and nothing else. /healthz is not the
// probe (R30), and HEAD is answered wherever GET is, since a probe may ask
// either way.
func TestRoutes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "the health probe", method: http.MethodGet, path: "/health", want: http.StatusOK},
		{name: "the health probe asked with HEAD", method: http.MethodHead, path: "/health", want: http.StatusOK},
		{name: "the path Cloud Run intercepts", method: http.MethodGet, path: "/healthz", want: http.StatusNotFound},
		{name: "an unknown path", method: http.MethodGet, path: "/unknown", want: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, http.NoBody)
			newServer().Handler.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Errorf("%s %s status = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
			}
		})
	}
}

// Every timeout of the server is set.
func TestEveryTimeoutIsSet(t *testing.T) {
	t.Parallel()

	server := newServer()
	for _, timeout := range []struct {
		name  string
		value time.Duration
	}{
		{name: "ReadHeaderTimeout", value: server.ReadHeaderTimeout},
		{name: "ReadTimeout", value: server.ReadTimeout},
		{name: "WriteTimeout", value: server.WriteTimeout},
		{name: "IdleTimeout", value: server.IdleTimeout},
	} {
		if timeout.value <= 0 {
			t.Errorf("%s = %v, want a positive duration", timeout.name, timeout.value)
		}
	}
}

// The port is the one PORT names, as Cloud Run sets it, or 8080 when PORT is
// empty; the host is empty, so every interface is served.
func TestListenAddr(t *testing.T) {
	for _, tc := range []struct {
		port string
		want string
	}{
		{port: "", want: ":8080"},
		{port: "9090", want: ":9090"},
	} {
		t.Run("PORT="+tc.port, func(t *testing.T) {
			t.Setenv("PORT", tc.port)

			if got := listenAddr(); got != tc.want {
				t.Errorf("listenAddr() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A stop lets the request in flight finish: the request the server is already
// handling when the stop begins still gets its answer, and serve then returns
// without an error.
func TestServeStopsGracefully(t *testing.T) {
	t.Parallel()

	arrived, release := make(chan struct{}), make(chan struct{})
	releaseHandler := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseHandler) // a test that fails early must not leave the handler waiting
	server := newServer()
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(arrived)
		<-release
		w.WriteHeader(http.StatusOK)
	})
	stopping := make(chan struct{})
	server.RegisterOnShutdown(func() { close(stopping) })

	listener := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- serve(ctx, server, listener) }()
	answered := make(chan error, 1)
	go func() { answered <- getOK(t.Context(), "http://"+listener.Addr().String()+"/") }()

	receive(t, arrived, "the request to reach the handler")
	cancel()
	receive(t, stopping, "the graceful stop to begin")
	releaseHandler()

	if err := receive(t, answered, "the answer to the request in flight"); err != nil {
		t.Errorf("the request in flight: %v, want its 200", err)
	}
	if err := receive(t, stopped, "serve() to return"); err != nil {
		t.Errorf("serve() error = %v, want nil", err)
	}
}

// A server that stops serving on its own is a failure: serve returns it at
// once, rather than waiting for a stop that nobody has asked for.
func TestServeReportsAServerThatStoppedOnItsOwn(t *testing.T) {
	t.Parallel()

	listener := listen(t)
	_ = listener.Close() // the server's first Accept fails, and nothing asked it to stop

	stopped := make(chan error, 1)
	go func() { stopped <- serve(t.Context(), newServer(), listener) }()

	if err := receive(t, stopped, "serve() to return"); !errors.Is(err, net.ErrClosed) {
		t.Errorf("serve() error = %v, want the closed listener reported", err)
	}
}

// A port that is already taken is reported as the failure it is, even when a
// stop is asked for at the same moment: run takes the port before it serves.
func TestRunReportsATakenPort(t *testing.T) {
	t.Parallel()

	taken := listen(t).Addr().String()
	for _, tc := range []struct {
		name      string
		stopAsked bool
	}{
		{name: "no stop asked for", stopAsked: false},
		{name: "a stop asked for at the same moment", stopAsked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, stop := context.WithCancel(t.Context())
			defer stop()
			if tc.stopAsked {
				stop()
			}

			if err := run(ctx, taken); !errors.Is(err, syscall.EADDRINUSE) {
				t.Errorf("run() error = %v, want the port reported in use", err)
			}
		})
	}
}

// Either signal a stop arrives by ends the context: the interrupt of a person
// at a terminal, and the termination Cloud Run sends before it takes an
// instance away. The signal is sent to this test binary itself.
func TestEitherStopSignalEndsTheContext(t *testing.T) {
	self, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("find this process: %v", err)
	}

	for _, sig := range []os.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			// A second listener for the same signal, so that a signal the
			// context fails to catch fails this test rather than ending the
			// binary that runs it. Cleanups run in reverse, so it outlives the
			// context's own.
			caught := make(chan os.Signal, 1)
			signal.Notify(caught, sig)
			t.Cleanup(func() { signal.Stop(caught) })

			ctx, stop := signalled()
			t.Cleanup(stop)

			if err := self.Signal(sig); err != nil {
				t.Fatalf("send %v: %v", sig, err)
			}
			receive(t, ctx.Done(), "the context to end")
		})
	}
}

// signalHelper names the variable that makes this test binary, started again
// by the test below, play a process on its way out instead of testing.
const signalHelper = "CANVAS_TEST_SIGNAL_HELPER"

// A second signal ends the process at once, whatever it is still doing: the
// first asked for a stop, and the drain can take seconds that nobody pressing
// Ctrl-C twice means to wait. The process is this test binary again, asked to
// stop and then stuck on its way out; a signal after that has to end it.
// Signals are sent until it ends or ten seconds pass, because the moment the
// first one stops being caught is the process's own.
func TestASecondSignalEndsTheProcess(t *testing.T) {
	if os.Getenv(signalHelper) == "1" {
		ctx, _ := signalled()
		fmt.Println("ready")
		<-ctx.Done()
		fmt.Println("stopping")
		time.Sleep(time.Minute) // a way out that does not end by itself
		return
	}
	t.Parallel()

	child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestASecondSignalEndsTheProcess$")
	child.Env = append(os.Environ(), signalHelper+"=1")
	said, err := child.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe() error = %v, want nil", err)
	}
	if err := child.Start(); err != nil {
		t.Fatalf("Start() error = %v, want nil", err)
	}
	lines := bufio.NewScanner(said)
	waitFor(t, lines, "ready")
	if err := child.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("the first signal: %v", err)
	}
	waitFor(t, lines, "stopping")

	ended := make(chan error, 1)
	go func() { ended <- child.Wait() }()
	for deadline := time.After(10 * time.Second); ; {
		_ = child.Process.Signal(os.Interrupt)
		select {
		case err := <-ended:
			if !endedBySignal(err) {
				t.Errorf("the process ended with %v, want it ended by the signal", err)
			}
			return
		case <-deadline:
			t.Fatal("the process still runs ten seconds after a second signal, want it ended at once")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// listen takes a free loopback port and holds it until the test ends, so that
// a test hands over a port it already has rather than one it hopes is free.
func listen(t *testing.T) net.Listener {
	t.Helper()

	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on a free port: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

// receive waits up to five seconds for what ch delivers, and fails the test,
// naming what it waited for, if nothing comes.
func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatalf("waited 5s for %s", what)
	}
	var zero T
	return zero
}

// getOK asks for url and says what went wrong unless the answer is 200. It
// runs beside the test, so it reports rather than failing the test itself.
func getOK(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

// endedBySignal says whether a process that ended with err was ended by a
// signal rather than by returning.
func endedBySignal(err error) bool {
	var exited *exec.ExitError
	if !errors.As(err, &exited) {
		return false
	}
	status, isWait := exited.Sys().(syscall.WaitStatus)
	return isWait && status.Signaled()
}

// waitFor reads what the process says until it says want.
func waitFor(t *testing.T, lines *bufio.Scanner, want string) {
	t.Helper()

	for lines.Scan() {
		if lines.Text() == want {
			return
		}
	}
	t.Fatalf("the process ended without saying %q", want)
}
