package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

// Producer publishes Protobuf-encoded canvas stroke events to AutoMQ.
type Producer struct {
	client *kgo.Client
	topic  string
}

func NewProducer(brokers []string, topic, saslUser, saslPass string) (*Producer, error) {
	auth := scram.Auth{User: saslUser, Pass: saslPass}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.SASL(auth.AsSha512Mechanism()),
		kgo.DefaultProduceTopic(topic),
	)
	if err != nil {
		return nil, fmt.Errorf("new kafka producer: %w", err)
	}
	return &Producer{client: client, topic: topic}, nil
}

// Publish sends a Protobuf-encoded payload to AutoMQ.
// sessionID is used as the partition key ([]byte) to guarantee that all strokes
// from one student session land in the same partition in chronological order —
// this is critical for correct OCR/LLM analysis downstream.
func (p *Producer) Publish(ctx context.Context, sessionID string, data []byte) error {
	record := &kgo.Record{
		Topic: p.topic,
		Key:   []byte(sessionID),
		Value: data,
	}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("kafka produce: %w", err)
	}
	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}
