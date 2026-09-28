// Command server runs canvas-api: one process on the one port Cloud Run gives
// it, answering the health probe and stopping gracefully when asked to.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// defaultPort is served when PORT is unset. Cloud Run always sets PORT, so
// this is the port of a run on a developer's machine.
const defaultPort = "8080"

// The server's timeouts, every one of them set: a client that stalls while
// sending its request, or while reading the answer, must not hold its
// connection for as long as it likes.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

// shutdownTimeout is how long a stop waits for the requests in flight. Cloud
// Run kills an instance ten seconds after its SIGTERM, so the wait ends a
// second sooner, while the process can still exit on its own.
const shutdownTimeout = 9 * time.Second

func main() {
	ctx, stop := signalled()
	err := run(ctx, listenAddr())
	stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "canvas-api: %v\n", err)
		os.Exit(1)
	}
}

// signalled is a context that ends when a stop is asked for: SIGTERM is how
// Cloud Run asks, SIGINT is Ctrl-C at a terminal. Once one has arrived, both
// go back to what they do by default, so a second signal ends the process at
// once instead of waiting out the drain.
func signalled() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	context.AfterFunc(ctx, stop)
	return ctx, stop
}

// listenAddr is every interface, on the port PORT names or on defaultPort.
func listenAddr() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return net.JoinHostPort("", port)
}

// run serves the service on addr until ctx ends. It takes the port before it
// serves, so a port that is already in use is reported as the failure it is,
// even when a stop arrives at the same moment.
func run(ctx context.Context, addr string) error {
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return serve(ctx, newServer(), listener)
}

// newServer is the service's HTTP server: its routes, and every timeout set.
func newServer() *http.Server {
	return &http.Server{
		Handler:           routes(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// serve runs server on listener until ctx ends, then stops it gracefully: the
// server takes no new connections, and the requests in flight get
// shutdownTimeout to finish.
func serve(ctx context.Context, server *http.Server, listener net.Listener) error {
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	select {
	case err := <-served:
		// Nothing asked the server to stop, so whatever stopped it is a failure.
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	// ctx has ended already, so the drain gets a deadline of its own, and it
	// starts now rather than when the signal arrived.
	drain, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(drain); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// routes is every path the service answers; any other is not found.
func routes() http.Handler {
	mux := http.NewServeMux()
	// /health rather than /healthz: Cloud Run's front end intercepts paths
	// that end in z (R30).
	mux.HandleFunc("GET /health", health)
	return mux
}

// health says the process is up and serving. It checks nothing else on
// purpose: a probe that reached for a dependency would turn that dependency's
// outage into restarts of this service.
func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
