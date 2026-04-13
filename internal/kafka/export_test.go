// Package kafka exposes internal helpers for white-box testing.
package kafka

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/infra/centrifugo"
)

// NewTestHintConsumer creates a HintConsumer with only the fields needed to
// test handle() — no live Kafka connection required.
func NewTestHintConsumer(pub centrifugo.Publisher, log *zap.Logger) *HintConsumer {
	return &HintConsumer{centrifugo: pub, log: log}
}

// HandleRecord exposes the unexported handle method for unit tests.
func (c *HintConsumer) HandleRecord(ctx context.Context, r *kgo.Record) error {
	return c.handle(ctx, r)
}
