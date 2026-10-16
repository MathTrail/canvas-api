package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// errStuckSocket is the fault of a listener that would not close.
var errStuckSocket = errors.New("the socket would not close")

// stuckListener is a listener that closes and then says it could not.
type stuckListener struct{ net.Listener }

func (l stuckListener) Close() error {
	_ = l.Listener.Close()
	return errStuckSocket
}

// A way out that fails for a reason of its own — a listener that would not
// close — is a failure, and is returned as one. Only a drain that ran out of
// its time is the stop that was asked for, cut short.
func TestAShutdownThatFailsIsNotADrainCutShort(t *testing.T) {
	t.Parallel()

	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	core, logs := observer.New(zapcore.InfoLevel)
	server := &Server{
		http:         &http.Server{Handler: http.NotFoundHandler(), ReadHeaderTimeout: time.Second},
		logger:       zap.New(core),
		drainTimeout: time.Second,
		listener:     stuckListener{listener},
	}

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- server.Run(ctx) }()

	// Stopped only once it is serving: a server stopped before it began
	// never gets to close its listener, and so never finds it stuck.
	answered(t, listener.Addr().String())
	cancel()

	select {
	case err := <-stopped:
		if !errors.Is(err, errStuckSocket) {
			t.Errorf("Run() error = %v, want the socket that would not close", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return within 5s of the context being cancelled")
	}
	if stopped := logs.FilterMessage("stopped").All(); len(stopped) != 0 {
		t.Errorf("stop lines = %v, want none for a way out that failed", stopped)
	}
}

// A server whose listener is gone stops serving on its own, which nobody asked
// for: Run returns that as a failure of serving, and no line says the server
// was asked to stop or stopped as asked.
func TestAServerThatStopsServingOnItsOwnFails(t *testing.T) {
	t.Parallel()

	var lc net.ListenConfig
	listener, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	if err = listener.Close(); err != nil {
		t.Fatalf("close the listener: %v", err)
	}
	core, logs := observer.New(zapcore.InfoLevel)
	server := &Server{
		http:         &http.Server{Handler: http.NotFoundHandler(), ReadHeaderTimeout: time.Second},
		logger:       zap.New(core),
		drainTimeout: time.Second,
		listener:     listener,
	}

	err = server.Run(t.Context())
	if err == nil || !strings.HasPrefix(err.Error(), "app: serve: ") || !errors.Is(err, net.ErrClosed) {
		t.Errorf("Run() error = %v, want a failure of serving on the closed listener", err)
	}
	for _, message := range []string{"shutdown requested", "stopped"} {
		if lines := logs.FilterMessage(message).All(); len(lines) != 0 {
			t.Errorf("%q lines = %v, want none for a server nobody asked to stop", message, lines)
		}
	}
}

// answered waits until the server at addr answers a request, whatever it
// answers.
func answered(t *testing.T, addr string) {
	t.Helper()

	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+addr+"/", http.NoBody)
		if err != nil {
			t.Fatalf("build a request: %v", err)
		}
		if response, err := client.Do(request); err == nil {
			_ = response.Body.Close()
			client.CloseIdleConnections()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no answer from %s within 5s", addr)
}
