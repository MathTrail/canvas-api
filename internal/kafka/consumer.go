package kafka

import (
	"context"
	"fmt"

	canvasv1 "github.com/mathtrail/contracts/gen/go/canvas/v1"
	"github.com/mathtrail/canvas-api/internal/infra/centrifugo"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// HintConsumer consumes HintEvent messages from AutoMQ and pushes them to the
// corresponding student's Centrifugo channel (canvas:{session_id}).
// It implements runner.Worker.
type HintConsumer struct {
	client     *kgo.Client
	centrifugo *centrifugo.Client
	log        *zap.Logger
}

func NewHintConsumer(
	brokers []string,
	topic, consumerGroup, saslUser, saslPass string,
	cClient *centrifugo.Client,
	log *zap.Logger,
) (*HintConsumer, error) {
	auth := scram.Auth{User: saslUser, Pass: saslPass}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.SASL(auth.AsSha512Mechanism()),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topic),
	)
	if err != nil {
		return nil, fmt.Errorf("new hint consumer: %w", err)
	}
	return &HintConsumer{client: client, centrifugo: cClient, log: log}, nil
}

// Start implements runner.Worker. It blocks until ctx is cancelled.
func (c *HintConsumer) Start(ctx context.Context) error {
	c.log.Info("hint consumer starting")
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			break
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			c.log.Error("kafka fetch error", zap.Any("errors", errs))
			continue
		}

		fetches.EachRecord(func(r *kgo.Record) {
			if err := c.handle(ctx, r); err != nil {
				c.log.Error("hint consumer: handle record",
					zap.String("topic", r.Topic),
					zap.Error(err),
				)
			}
		})
	}

	c.log.Info("hint consumer stopping, leaving group")
	c.client.LeaveGroup()
	return nil
}

func (c *HintConsumer) handle(ctx context.Context, r *kgo.Record) error {
	var hint canvasv1.HintEvent
	if err := proto.Unmarshal(r.Value, &hint); err != nil {
		return fmt.Errorf("unmarshal HintEvent: %w", err)
	}

	// Re-encode for the Centrifugo publish payload so the client receives
	// binary Protobuf data which centrifuge-js decodes with HintEvent.fromBinary().
	data, err := proto.Marshal(&hint)
	if err != nil {
		return fmt.Errorf("re-marshal HintEvent: %w", err)
	}

	channel := "canvas:" + hint.SessionId
	if err := c.centrifugo.Publish(ctx, channel, data); err != nil {
		return fmt.Errorf("centrifugo publish hint to %s: %w", channel, err)
	}

	c.log.Info("hint pushed", zap.String("session_id", hint.SessionId), zap.String("hint_type", hint.HintType))
	return nil
}

// Close releases the Kafka client.
func (c *HintConsumer) Close() {
	c.client.Close()
}
