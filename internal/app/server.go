package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Server is the HTTP server and its graceful shutdown.
type Server struct {
	http   *http.Server
	logger *zap.Logger
	// drainTimeout is how long a shutdown waits for the requests in flight:
	// the server's share of the way out, not the whole of it.
	drainTimeout time.Duration

	// mu guards listener: Run binds it on one goroutine while Addr may be
	// read from another, which is how a caller finds out which port the
	// kernel picked.
	mu       sync.RWMutex
	listener net.Listener
}

// NewServer builds the server from a wired container.
func NewServer(c *Container) *Server {
	return &Server{
		logger:       c.Logger,
		drainTimeout: c.Config.DrainTimeout(),
		http: &http.Server{
			// Empty host, so the server listens on every interface. The
			// configuration guarantees a bare port number, and JoinHostPort is
			// what keeps the address well formed if that ever stops being true.
			Addr:              net.JoinHostPort("", c.Config.Port),
			Handler:           c.Router,
			ReadHeaderTimeout: c.Config.ReadHeaderTimeout,
			ReadTimeout:       c.Config.ReadTimeout,
			WriteTimeout:      c.Config.WriteTimeout,
			IdleTimeout:       c.Config.IdleTimeout,
		},
	}
}

// Listen binds the port without serving on it yet. Run binds it itself when
// nothing did, but the process calls Listen first: binding early is how a port
// that is already taken is reported as a failed start, before the service says
// it has started, rather than swallowed by the goroutine inside Run. A test
// calls it too, when it needs the real address before the server starts.
func (s *Server) Listen(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return nil
	}
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", s.http.Addr)
	if err != nil {
		return fmt.Errorf("app: listen on %s: %w", s.http.Addr, err)
	}
	s.listener = listener
	return nil
}

// Addr is the address the server is bound to, empty before Listen. It is safe
// to call while the server is starting or serving.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Run serves until ctx is cancelled, then shuts down gracefully. Shutdown is
// driven by the context rather than by a signal handler of its own, so that
// every part of the process stops for the same reason at the same moment.
//
// It returns only once the goroutine it serves on has finished, so every line
// Run itself logs is written by then. A request still being handled when the
// drain's share of the way out is spent is cut off rather than waited for, and
// that is still the stop that was asked for: the line saying so is a warning,
// and Run returns no error. A failure is what nobody asked for — a server that
// stopped serving on its own.
func (s *Server) Run(ctx context.Context) error {
	if err := s.Listen(ctx); err != nil {
		return err
	}

	s.mu.RLock()
	listener := s.listener
	s.mu.RUnlock()

	// The port is bound already, so this is true before serving starts, and
	// written here it cannot land after Run has returned.
	s.logger.Info("listening", zap.String("addr", listener.Addr().String()))
	served := make(chan error, 1)
	go func() { served <- s.http.Serve(listener) }()

	select {
	case err := <-served:
		// Nothing had asked it to stop, so whatever it returned is a failure.
		return fmt.Errorf("app: serve: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutdown requested")
	}

	// A fresh context, so the drain's deadline starts now rather than at the
	// moment the signal arrived and cancelled the parent. It is the drain's
	// share of the way out: whoever closes what the server ran on needs the
	// rest before the platform stops the process.
	shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.drainTimeout)
	defer cancel()

	drained := s.http.Shutdown(shutdown)
	if drained != nil {
		_ = s.http.Close() // the deadline passed: drop what is left rather than hang
	}
	// Shutdown and Close both make Serve return, even one that had not begun
	// yet, so this wait is short.
	<-served

	switch {
	case errors.Is(drained, context.DeadlineExceeded):
		s.logger.Warn("stopped", zap.Bool("cut_short", true))
		return nil
	case drained != nil:
		return fmt.Errorf("app: graceful shutdown: %w", drained)
	}
	s.logger.Info("stopped")
	return nil
}
