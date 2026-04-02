package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server wraps the HTTP server with graceful shutdown.
type Server struct {
	httpServer      *http.Server
	logger          *zap.Logger
	shutdownTimeout time.Duration
}

// NewServer creates a Server from the wired Container.
func NewServer(container *Container) *Server {
	return &Server{
		logger:          container.Logger,
		shutdownTimeout: container.Config.ShutdownTimeout,
		httpServer: &http.Server{
			Addr:              fmt.Sprintf(":%s", container.Config.ServerPort),
			Handler:           container.Router,
			ReadHeaderTimeout: container.Config.ReadHeaderTimeout,
			ReadTimeout:       container.Config.ReadTimeout,
			WriteTimeout:      container.Config.WriteTimeout,
			IdleTimeout:       container.Config.IdleTimeout,
		},
	}
}

// Run starts the HTTP listener and blocks until ctx is cancelled or the server
// fails. Shutdown is triggered by ctx cancellation, not by OS signals directly.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("listening", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
		s.logger.Info("shutdown signal received")
	}

	// Fresh context so the shutdown deadline starts now, not at signal time.
	shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()

	s.logger.Info("shutting down gracefully", zap.Duration("timeout", s.shutdownTimeout))

	if err := s.httpServer.Shutdown(shutCtx); err != nil {
		_ = s.httpServer.Close() // force-close remaining connections
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	s.logger.Info("server shut down gracefully")
	return nil
}
