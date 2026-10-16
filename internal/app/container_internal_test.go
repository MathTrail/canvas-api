package app

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/MathTrail/canvas-api/internal/config"
)

// Close runs every closer, the last registered first, so that a resource is
// closed before whatever it was built from. One that fails is logged as a
// failure, and the ones after it still run rather than being left open.
func TestCloseRunsEveryCloserLastFirstAndLogsTheOneThatFails(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zapcore.InfoLevel)
	stuck := errors.New("the client would not close")
	var ran []string
	c := &Container{Logger: zap.New(core)}
	c.onClose(closing(&ran, "first", nil))
	c.onClose(closing(&ran, "second", stuck))
	c.onClose(closing(&ran, "third", nil))

	c.Close(t.Context())
	if want := []string{"third", "second", "first"}; !slices.Equal(ran, want) {
		t.Errorf("closed %q, want %q: every closer, the last registered first", ran, want)
	}
	failed := logs.FilterMessage("close failed").All()
	if len(failed) != 1 || failed[0].Level != zapcore.ErrorLevel || failed[0].ContextMap()["error"] != stuck.Error() {
		t.Errorf("close failed lines = %v, want one error naming %q", failed, stuck)
	}
}

// A start that fails halfway releases what it had built, the last built first,
// and builds nothing after the piece that failed. The close gets a context of
// its own: the start may have failed because its context ended.
func TestAStartThatFailsHalfwayClosesWhatItBuilt(t *testing.T) {
	t.Parallel()

	refused := errors.New("the third piece could not be built")
	var ran []string
	var closedLive bool
	built := func(name string) part {
		return func(_ context.Context, c *Container) error {
			c.onClose(func(ctx context.Context) error {
				closedLive = closedLive || ctx.Err() == nil
				ran = append(ran, name)
				return nil
			})
			return nil
		}
	}
	never := func(context.Context, *Container) error {
		t.Error("a piece after the one that failed was built")
		return nil
	}
	failing := func(context.Context, *Container) error { return refused }

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // the start was stopped, and its context is over
	c, err := build(ctx, &config.Config{ShutdownTimeout: config.DefaultShutdownTimeout}, zaptest.NewLogger(t),
		[]part{built("first"), built("second"), failing, never})

	if !errors.Is(err, refused) || c != nil {
		t.Fatalf("build() = %v, %v, want no container and the piece's error", c, err)
	}
	if want := []string{"second", "first"}; !slices.Equal(ran, want) {
		t.Errorf("closed %q, want %q: what was built, the last first", ran, want)
	}
	if !closedLive {
		t.Error("the close got a context that had ended, want one of its own")
	}
}

// A start that succeeds closes nothing until it is asked to.
func TestAStartThatSucceedsClosesNothing(t *testing.T) {
	t.Parallel()

	var ran []string
	c, err := build(t.Context(), &config.Config{ShutdownTimeout: config.DefaultShutdownTimeout}, zaptest.NewLogger(t),
		[]part{func(_ context.Context, c *Container) error {
			c.onClose(closing(&ran, "only", nil))
			return nil
		}})
	if err != nil {
		t.Fatalf("build() error = %v, want nil", err)
	}
	if len(ran) != 0 {
		t.Errorf("closed %q before Close, want nothing", ran)
	}
	c.Close(t.Context())
	if want := []string{"only"}; !slices.Equal(ran, want) {
		t.Errorf("closed %q, want %q", ran, want)
	}
}

// closing is a closer that writes its name down when it runs, and fails with
// err.
func closing(ran *[]string, name string, err error) func(context.Context) error {
	return func(context.Context) error {
		*ran = append(*ran, name)
		return err
	}
}
