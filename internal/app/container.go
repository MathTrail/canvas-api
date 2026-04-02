package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/config"
	"github.com/mathtrail/canvas-api/internal/handlers"
	centrifugoclient "github.com/mathtrail/canvas-api/internal/infra/centrifugo"
	"github.com/mathtrail/canvas-api/internal/kafka"
	"github.com/mathtrail/canvas-api/internal/runner"
	httpserver "github.com/mathtrail/canvas-api/internal/transport/http"
)

// Container holds the dependencies consumed by the Server.
// Internal wiring is kept as local variables in NewContainer and is not exposed.
type Container struct {
	Config  *config.Config
	Logger  *zap.Logger
	Router  *gin.Engine
	Workers []runner.Worker
	stop    func()
}

// NewContainer creates and wires all application dependencies.
// It returns an error instead of panicking so that the caller can
// handle failures gracefully.
func NewContainer(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Container, error) {
	cClient := centrifugoclient.NewClient(cfg.CentrifugoURL, cfg.CentrifugoAPIKey)

	producer, err := kafka.NewProducer(
		cfg.AutoMQBrokers,
		cfg.StrokeTopic,
		cfg.KafkaSASLUsername,
		cfg.KafkaSASLPassword,
	)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}

	consumer, err := kafka.NewHintConsumer(
		cfg.AutoMQBrokers,
		cfg.HintTopic,
		cfg.KafkaConsumerGroup,
		cfg.KafkaSASLUsername,
		cfg.KafkaSASLPassword,
		cClient,
		logger,
	)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("hint consumer: %w", err)
	}

	strokeHandler := handlers.NewStrokeHandler(producer, logger)
	tokenHandler := handlers.NewTokenHandler(cfg.CentrifugoHMACKey, logger)
	healthHandler := httpserver.NewHealthHandler(cClient)
	router := httpserver.NewRouter(strokeHandler, tokenHandler, healthHandler, cfg, logger)

	return &Container{
		Config:  cfg,
		Logger:  logger,
		Router:  router,
		Workers: []runner.Worker{consumer},
		// Closes resources in reverse init order.
		stop: func() {
			consumer.Close()
			producer.Close()
		},
	}, nil
}

// Close releases resources held by the container.
// Call once after RunWorkers and the HTTP server have both stopped.
// ctx is used as a deadline for the shutdown: if resources take too long to
// close, the method returns and logs a warning rather than blocking forever.
func (c *Container) Close(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.stop()
	}()
	select {
	case <-done:
	case <-ctx.Done():
		c.Logger.Warn("container close timed out", zap.Error(ctx.Err()))
	}
}
