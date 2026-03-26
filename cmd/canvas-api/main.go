package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/mathtrail/canvas-api/internal/config"
	centrifugoclient "github.com/mathtrail/canvas-api/internal/infra/centrifugo"
	"github.com/mathtrail/canvas-api/internal/handlers"
	"github.com/mathtrail/canvas-api/internal/kafka"
	"github.com/mathtrail/canvas-api/internal/middleware"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("load config", zap.Error(err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Infrastructure
	cClient := centrifugoclient.NewClient(cfg.CentrifugoURL, cfg.CentrifugoAPIKey)

	producer, err := kafka.NewProducer(cfg.AutoMQBrokers, cfg.StrokeTopic, cfg.KafkaSASLUsername, cfg.KafkaSASLPassword)
	if err != nil {
		log.Fatal("kafka producer", zap.Error(err))
	}
	defer producer.Close()

	consumer, err := kafka.NewHintConsumer(
		cfg.AutoMQBrokers,
		cfg.HintTopic,
		cfg.KafkaConsumerGroup,
		cfg.KafkaSASLUsername,
		cfg.KafkaSASLPassword,
		cClient,
		log,
	)
	if err != nil {
		log.Fatal("hint consumer", zap.Error(err))
	}
	defer consumer.Close()

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.Get("/health", handlers.Health)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.OryKratosURL))
		r.Get("/api/canvas/token", handlers.Token(cfg.CentrifugoHMACKey))
		r.Post("/api/canvas/strokes", handlers.Strokes(producer))
	})

	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	g, ctx := errgroup.WithContext(ctx)

	// HTTP server
	g.Go(func() error {
		log.Info("canvas-api starting", zap.String("addr", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Hint consumer
	g.Go(func() error {
		return consumer.Run(ctx)
	})

	// Graceful shutdown
	g.Go(func() error {
		<-ctx.Done()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		return srv.Shutdown(shutCtx)
	})

	if err := g.Wait(); err != nil {
		log.Error("canvas-api exited with error", zap.Error(err))
	}
}
