// Package app wires the service together and runs it.
//
// The container is hand-written on purpose: there is no DI framework, the
// order of construction is the order of the code, and the order of shutdown is
// its reverse.
package app

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"github.com/MathTrail/canvas-api/internal/config"
	httpserver "github.com/MathTrail/canvas-api/internal/transport/http"
)

// Container holds everything the process needs while it runs, and knows how to
// close it again. Every part is built in one place, NewContainer, and released
// in one, Close.
type Container struct {
	Config *config.Config
	Logger *zap.Logger
	Router http.Handler

	// closers run in reverse order of registration, so that a resource is
	// always closed before whatever it was built from.
	closers []func(context.Context) error
}

// part builds one piece of the container, and registers with onClose whatever
// that piece has to release.
type part func(ctx context.Context, c *Container) error

// parts are the pieces of the service, in the order they are built: a piece
// may use any piece before it.
var parts = []part{
	routes,
}

// NewContainer builds the service from its configuration. A piece that cannot
// be built stops the start, and whatever was built before it is closed again.
func NewContainer(ctx context.Context, cfg *config.Config, log *zap.Logger) (*Container, error) {
	return build(ctx, cfg, log, parts)
}

// build builds the container from the pieces given. It is NewContainer with the
// pieces as an argument, so that a test can make one of them fail.
func build(ctx context.Context, cfg *config.Config, log *zap.Logger, pieces []part) (_ *Container, err error) {
	c := &Container{Config: cfg, Logger: log}
	defer func() {
		if err == nil {
			return
		}
		// Detached, with the time a close is given: construction may have
		// failed because its context ended, and a close handed that context
		// would give up before it began.
		closing, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.CloseTimeout())
		defer cancel()
		c.Close(closing)
	}()

	for _, piece := range pieces {
		if err := piece(ctx, c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// routes builds the HTTP surface. It holds nothing to release.
func routes(_ context.Context, c *Container) error {
	c.Router = httpserver.NewRouter(c.Logger)
	return nil
}

// onClose registers what Close has to release.
func (c *Container) onClose(closer func(context.Context) error) {
	c.closers = append(c.closers, closer)
}

// Close releases everything the container holds, the last built first. One
// that fails is logged, and the ones after it still run rather than being left
// open.
func (c *Container) Close(ctx context.Context) {
	for i := len(c.closers) - 1; i >= 0; i-- {
		if err := c.closers[i](ctx); err != nil {
			c.Logger.Error("close failed", zap.Error(err))
		}
	}
}
