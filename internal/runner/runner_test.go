package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mathtrail/canvas-api/internal/runner"
	runnermocks "github.com/mathtrail/canvas-api/internal/runner/mocks"
)

func TestRunGroup_AllWorkersComplete(t *testing.T) {
	// errgroup.WithContext wraps the ctx we pass, so workers receive a derived
	// context — use mock.Anything to avoid brittle pointer equality checks.
	w1 := runnermocks.NewMockWorker(t)
	w2 := runnermocks.NewMockWorker(t)
	w1.EXPECT().Start(mock.Anything).Return(nil)
	w2.EXPECT().Start(mock.Anything).Return(nil)

	err := runner.RunGroup(context.Background(), w1, w2)
	assert.NoError(t, err)
}

func TestRunGroup_OneWorkerErrors_CancelsOthers(t *testing.T) {
	boom := errors.New("worker exploded")

	w1 := runnermocks.NewMockWorker(t)
	w2 := runnermocks.NewMockWorker(t)

	// w1 immediately returns an error; the errgroup cancels the shared ctx so
	// w2 receives a cancelled context and returns.
	w1.EXPECT().Start(mock.Anything).Return(boom)
	w2.EXPECT().Start(mock.Anything).RunAndReturn(func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})

	err := runner.RunGroup(context.Background(), w1, w2)
	require.Error(t, err)
	assert.Equal(t, boom, err)
}

func TestRunGroup_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w := runnermocks.NewMockWorker(t)
	// Worker still starts — it receives a derived (already-cancelled) ctx.
	w.EXPECT().Start(mock.Anything).Return(nil)

	err := runner.RunGroup(ctx, w)
	assert.NoError(t, err)
}

func TestRunGroup_NoWorkers(t *testing.T) {
	err := runner.RunGroup(context.Background())
	assert.NoError(t, err)
}
