package runner

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Worker is a long-running background process that terminates when ctx is cancelled.
type Worker interface {
	Start(ctx context.Context) error
}

// RunGroup starts each worker in its own goroutine under a shared errgroup.
// The group context is cancelled if any worker returns a non-nil error.
func RunGroup(ctx context.Context, workers ...Worker) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, w := range workers {
		w := w
		g.Go(func() error { return w.Start(ctx) })
	}
	return g.Wait()
}
