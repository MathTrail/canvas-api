package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/MathTrail/canvas-api/internal/config"
	"github.com/MathTrail/canvas-api/internal/version"
)

// run must stop because its context was cancelled — the same way a signal
// reaches it — and it must stop without an error.
func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	ownEnvironment(t)
	port := freePort(t)
	t.Setenv("PORT", port)
	t.Setenv("CANVAS_LOG_LEVEL", "error")
	t.Setenv("CANVAS_SHUTDOWN_TIMEOUT", "2s")

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- run(ctx) }()

	// Cancel only once the service is actually listening, so that a pass
	// means "it started and then stopped" rather than "it never started".
	waitUntilListening(t, port)
	cancel()

	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("run() error = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run() did not return within 10s of the context being cancelled")
	}
}

// Once it serves, the service says which build it is in the line the
// delivery's checklist reads a revision's release by: JSON a collector reads
// the severity of, with the message startup, the version and the commit.
func TestTheStartupLineNamesTheBuild(t *testing.T) {
	ownEnvironment(t)
	was := [3]string{version.Version, version.Commit, version.Date}
	version.Version, version.Commit, version.Date = "v1.2.3", "abc1234", "2026-10-12T00:00:00Z"
	t.Cleanup(func() { version.Version, version.Commit, version.Date = was[0], was[1], was[2] })
	port := freePort(t)
	t.Setenv("PORT", port)
	t.Setenv("CANVAS_SHUTDOWN_TIMEOUT", "2s")

	lines := capturedStdout(t)
	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- run(ctx) }()
	waitUntilListening(t, port)
	cancel()
	if err := <-stopped; err != nil {
		t.Fatalf("run() error = %v, want nil", err)
	}

	var startup map[string]any
	for line := range bytes.Lines(lines()) {
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("a line of the log is not JSON: %v; line = %s", err, line)
		}
		if entry["message"] == "startup" {
			startup = entry
		}
	}
	if startup == nil {
		t.Fatal("no startup line in the log")
	}
	for key, want := range map[string]any{
		"severity": "INFO",
		"version":  "v1.2.3",
		"commit":   "abc1234",
		"date":     "2026-10-12T00:00:00Z",
		"port":     port,
		"env":      "local",
		"deployed": false,
	} {
		if startup[key] != want {
			t.Errorf("startup[%q] = %v, want %v", key, startup[key], want)
		}
	}
}

// ownEnvironment clears every variable the service reads, so that what the
// shell running the tests exported — a console log format, a K_SERVICE — does
// not decide the outcome. An empty value counts as unset.
func ownEnvironment(t *testing.T) {
	t.Helper()

	for _, name := range []string{
		"PORT", "K_SERVICE", "CANVAS_ENV", "CANVAS_LOG_LEVEL", "CANVAS_LOG_FORMAT",
		"CANVAS_HTTP_READ_HEADER_TIMEOUT", "CANVAS_HTTP_READ_TIMEOUT", "CANVAS_HTTP_WRITE_TIMEOUT",
		"CANVAS_HTTP_IDLE_TIMEOUT", "CANVAS_SHUTDOWN_TIMEOUT",
	} {
		t.Setenv(name, "")
	}
}

// capturedStdout sends what the process writes to stdout into a pipe until the
// returned function is called, which hands back everything written.
func capturedStdout(t *testing.T) func() []byte {
	t.Helper()

	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = read.Close() })
	written := make(chan []byte, 1)
	go func() {
		all, _ := io.ReadAll(read)
		written <- all
	}()
	stdout := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = stdout })
	return func() []byte {
		os.Stdout = stdout
		_ = write.Close()
		return <-written
	}
}

// A failure is reported once: in the log, as one line a collector reads the
// severity of, and marked as written so that the process does not print it a
// second time on its way out.
func TestAFailureIsReportedOnce(t *testing.T) {
	ownEnvironment(t)
	var lc net.ListenConfig
	taken, err := lc.Listen(t.Context(), "tcp", ":0")
	if err != nil {
		t.Fatalf("listen for a port to take: %v", err)
	}
	t.Cleanup(func() { _ = taken.Close() })
	addr, ok := taken.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address = %T, want *net.TCPAddr", taken.Addr())
	}
	t.Setenv("PORT", strconv.Itoa(addr.Port))

	lines := capturedStdout(t)
	err = run(t.Context())
	written := lines()

	if !errors.Is(err, errLogged) {
		t.Fatalf("run() error = %v, want the taken port reported and marked as logged", err)
	}
	var failures, starts int
	for line := range bytes.Lines(written) {
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("a line of the log is not JSON: %v; line = %s", err, line)
		}
		switch {
		case entry["severity"] == "ERROR" && entry["message"] == "startup":
			failures++
		case entry["severity"] == "ERROR":
			t.Errorf("an error line = %s, want the failure named startup: the start is what failed", line)
		case entry["message"] == "startup":
			starts++
		}
	}
	// A port the process never held is a start that failed, written once, and
	// no line says the service started on it.
	if failures != 1 || starts != 0 {
		t.Errorf("the log has %d failed starts and %d announced ones, want 1 and 0:\n%s", failures, starts, written)
	}
}

// A configuration it cannot satisfy is reported, not worked around: nothing is
// started and the error says which variable is wrong.
func TestRunRefusesBadConfiguration(t *testing.T) {
	ownEnvironment(t)
	t.Setenv("PORT", "eighty")

	err := run(t.Context())
	if !errors.Is(err, config.ErrInvalid) {
		t.Fatalf("run() error = %v, want it to wrap config.ErrInvalid", err)
	}
	// There is no log yet to have written it, so it is the process that
	// reports it.
	if errors.Is(err, errLogged) {
		t.Errorf("run() error = %v, want it left for the process to print", err)
	}
}

// The process exits with 1 whenever run failed, and prints a failure only when
// nothing else has reported it: the one the log recorded stays a single line,
// and the one from before there was a log has nowhere else to go.
func TestAFailureIsPrintedOnlyWhenTheLogDidNotRecordIt(t *testing.T) {
	t.Parallel()

	refused := fmt.Errorf("%w: PORT must be a number from 1 to 65535", config.ErrInvalid)
	for _, tc := range []struct {
		name    string
		err     error
		code    int
		printed string
	}{
		{name: "a stop that was asked for", err: nil, code: 0, printed: ""},
		{name: "a failure from before there was a log", err: refused, code: 1,
			printed: "canvas-api: " + refused.Error() + "\n"},
		{name: "a failure the log recorded", err: logged(zap.NewNop(), "startup", refused), code: 1, printed: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			if code := exitCode(&stderr, tc.err); code != tc.code {
				t.Errorf("exitCode() = %d, want %d", code, tc.code)
			}
			if printed := stderr.String(); printed != tc.printed {
				t.Errorf("printed %q, want %q", printed, tc.printed)
			}
		})
	}
}

// A stop asked for while the service is still being built ends the process as
// the stop it is, with nothing written as an error. A failure of the build is
// a failure even when a stop arrived beside it, and so is a cancellation that
// nobody asked this process for.
func TestAStopDuringTheStartIsNoFailure(t *testing.T) {
	t.Parallel()

	stopped, stop := context.WithCancel(t.Context())
	stop()
	cutShort := fmt.Errorf("app: build the router: %w", context.Canceled)
	broken := errors.New("app: a piece could not be built")

	for _, tc := range []struct {
		name       string
		ctx        context.Context
		err        error
		failed     bool
		errorLines int
	}{
		{name: "the build cut short by the stop", ctx: stopped, err: cutShort, failed: false, errorLines: 0},
		{name: "a build that failed while a stop arrived", ctx: stopped, err: broken, failed: true, errorLines: 1},
		{name: "a cancellation nobody asked for", ctx: t.Context(), err: cutShort, failed: true, errorLines: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			core, logs := observer.New(zapcore.DebugLevel)
			err := startupFailed(tc.ctx, zap.New(core), tc.err)

			if errors.Is(err, errLogged) != tc.failed {
				t.Errorf("startupFailed() = %v, want a failure: %v", err, tc.failed)
			}
			if lines := logs.FilterLevelExact(zapcore.ErrorLevel).Len(); lines != tc.errorLines {
				t.Errorf("error lines = %d, want %d", lines, tc.errorLines)
			}
			if !tc.failed && err != nil {
				t.Errorf("startupFailed() = %v, want nil for the stop that was asked for", err)
			}
		})
	}
}

// Either signal a stop arrives by ends the context: the interrupt of a person
// at a terminal, and the termination a platform sends before it takes an
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
			select {
			case <-ctx.Done():
			case <-time.After(10 * time.Second):
				t.Fatalf("the context is still live 10s after %v, want it ended", sig)
			}
		})
	}
}

// freePort asks the kernel for a port and gives it straight back, so the test
// binds a port nobody else is using rather than a number somebody guessed.
func freePort(t *testing.T) string {
	t.Helper()

	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for a free port: %v", err)
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address = %T, want *net.TCPAddr", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close the probe listener: %v", err)
	}
	return strconv.Itoa(addr.Port)
}

func waitUntilListening(t *testing.T, port string) {
	t.Helper()

	var dialer net.Dialer
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := dialer.DialContext(t.Context(), "tcp", net.JoinHostPort("127.0.0.1", port))
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("nothing is listening on port %s after 10s", port)
}

// signalHelper names the variable that makes this test binary, started again
// by the test below, play a process on its way out instead of testing.
const signalHelper = "CANVAS_TEST_SIGNAL_HELPER"

// A second signal ends the process at once, whatever it is still doing: the
// first asked for a stop, and the way out can take seconds that nobody
// pressing Ctrl-C twice means to wait. The process is this test binary again,
// asked to stop and then stuck on its way out; a signal after that has to end
// it. Signals are sent until it ends or ten seconds pass, because the moment
// the first one stops being caught is the process's own.
func TestASecondSignalEndsTheProcess(t *testing.T) {
	if os.Getenv(signalHelper) == "1" {
		ctx, _ := signalled()
		fmt.Println("ready")
		<-ctx.Done()
		fmt.Println("stopping")
		time.Sleep(time.Minute) // a way out that does not end by itself
		return
	}

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
