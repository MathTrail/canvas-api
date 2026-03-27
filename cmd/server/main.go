package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/mathtrail/canvas-api/internal/app"
	"github.com/mathtrail/canvas-api/internal/config"
	applogger "github.com/mathtrail/canvas-api/internal/logger"
	"github.com/mathtrail/canvas-api/internal/observability"
	"github.com/mathtrail/canvas-api/internal/runner"
)

func main() {
	// 1. Single point of config and logger creation.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger := applogger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// 2. Root context: cancelled on SIGINT or SIGTERM.
	// Created first so it can be passed into every subsystem.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 3. Observability stack (tracing, metrics, profiling).
	obs := observability.New(cfg, logger)
	if err := obs.Init(ctx); err != nil {
		logger.Fatal("failed to initialize observability", zap.Error(err))
	}
	// Shutdown context is created at exit time so the deadline starts
	// only when the process is actually terminating.
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		obs.Shutdown(shutCtx)
	}()

	// 4. DI container (Kafka, Centrifugo, handlers, router).
	container, err := app.NewContainer(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("failed to initialize application", zap.Error(err))
	}
	defer container.Close()

	logger.Info("starting canvas-api", zap.String("port", cfg.ServerPort))

	// 5. Run background workers and HTTP server under a shared errgroup so that:
	//    - container.Close() (deferred above) only runs after both have exited;
	//    - if any component fails, gCtx is cancelled and the others begin graceful shutdown.
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return runner.RunGroup(gCtx, container.Workers...)
	})

	srv := app.NewServer(container)
	g.Go(func() error {
		return srv.Run(gCtx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("application stopped with error", zap.Error(err))
	}
	logger.Info("canvas-api stopped gracefully")
}
