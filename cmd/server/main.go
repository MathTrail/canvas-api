// Command server runs canvas-api: one process, one port, and no
// state of its own between requests.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/MathTrail/canvas-api/internal/app"
	"github.com/MathTrail/canvas-api/internal/config"
	"github.com/MathTrail/canvas-api/internal/logger"
	"github.com/MathTrail/canvas-api/internal/version"
)

// main owns the two things a process owns and a function should not: the
// signals it listens for and the code it exits with. Everything else is run,
// which stops because its context was cancelled and says so by returning.
func main() {
	ctx, stop := signalled()
	err := run(ctx)
	stop()

	// The exit is the last statement of the process on purpose: os.Exit runs
	// no deferred call, so nothing may be left to one after this point.
	os.Exit(exitCode(os.Stderr, err))
}

// exitCode is the code the process exits with once run has returned err, and
// it prints the one failure nobody else will have: a failure the log has
// recorded is not printed again, and one from before there was a log has
// nowhere else to go.
func exitCode(stderr io.Writer, err error) int {
	if err == nil {
		return 0
	}
	if !errors.Is(err, errLogged) {
		fmt.Fprintf(stderr, "canvas-api: %v\n", err)
	}
	return 1
}

// signalled is a context that ends when a stop is asked for, by either of the
// signals that ask for one. One signal asks; a second ends the process at once.
// Once the first has arrived the signals go back to what they do by default,
// rather than being caught for the whole of the way out, which can take
// seconds nobody pressing Ctrl-C twice means to wait.
func signalled() (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	context.AfterFunc(ctx, stop)
	return ctx, stop
}

// errLogged marks a failure the log has already recorded, so that it is
// reported once.
var errLogged = errors.New("logged")

// logged writes a failure to the log under the name of the moment it happened
// at, and marks it as written.
func logged(log *zap.Logger, moment string, err error) error {
	log.Error(moment, zap.Error(err))
	return fmt.Errorf("%w: %s: %w", errLogged, moment, err)
}

// startupFailed is what run returns when the service could not be built. A
// build cut short by the stop that was asked for is that stop, not a failure:
// a platform may take an instance away while it is still starting, and doing
// so is no fault of the instance. Anything else is a failure, and is written
// down as one.
func startupFailed(ctx context.Context, log *zap.Logger, err error) error {
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		log.Info("shutdown", zap.Bool("started", false))
		return nil
	}
	return logged(log, "startup", err)
}

// run starts the service, serves until ctx is cancelled and closes it again.
// Every deferred call runs on the way out, which is why nothing here calls
// os.Exit.
func run(ctx context.Context) error {
	// The configuration is read before anything else exists, so a service that
	// is configured wrongly says so and exits instead of half-starting.
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		return err
	}
	defer func() { _ = log.Sync() }()

	// The web framework keeps its mode in a package variable, so it belongs
	// where the process is set up rather than inside whatever builds a router.
	gin.SetMode(gin.ReleaseMode)

	container, err := app.NewContainer(ctx, cfg, log)
	if err != nil {
		return startupFailed(ctx, log, err)
	}
	// Deferred, so that the container is closed on every way out of here. It
	// gets the share of the way out the server's drain always leaves it.
	defer func() {
		closing, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.CloseTimeout())
		defer cancel()
		container.Close(closing)
	}()

	// The port is taken before the service says it has started, so that a port
	// already in use is the failed start it is, and the startup line never
	// names a port the process does not hold.
	server := app.NewServer(container)
	if err := server.Listen(ctx); err != nil {
		return startupFailed(ctx, log, err)
	}

	// The delivery's checklist finds the release a revision runs by this line,
	// its message and its version and commit.
	log.Info("startup",
		zap.String("version", version.Version),
		zap.String("commit", version.Commit),
		zap.String("date", version.Date),
		zap.String("port", cfg.Port),
		zap.String("env", cfg.Env),
		zap.Bool("deployed", cfg.Deployed()),
	)

	// Everything that runs for the life of the process shares one group: when
	// one part fails, the others are told to stop, and the deferred close runs
	// only after all of them have.
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error { return server.Run(groupCtx) })

	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return logged(log, "shutdown", err)
	}
	log.Info("shutdown")
	return nil
}
