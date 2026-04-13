package kafka_test

import (
	"context"
	"errors"
	"testing"

	canvasv1 "github.com/mathtrail/contracts/gen/go/canvas/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	centrifugomocks "github.com/mathtrail/canvas-api/internal/infra/centrifugo/mocks"
	"github.com/mathtrail/canvas-api/internal/kafka"
)

func makeHintRecord(t *testing.T, sessionID string) *kgo.Record {
	t.Helper()
	hint := &canvasv1.HintEvent{SessionId: sessionID, HintType: "arrow"}
	b, err := proto.Marshal(hint)
	require.NoError(t, err)
	return &kgo.Record{Topic: "canvas.hints", Value: b}
}

func TestHintConsumer_Handle_ValidEvent(t *testing.T) {
	pub := centrifugomocks.NewMockPublisher(t)
	pub.EXPECT().
		Publish(mock.Anything, "canvas:session-1", mock.Anything).
		Return(nil)

	c := kafka.NewTestHintConsumer(pub, zap.NewNop())
	r := makeHintRecord(t, "session-1")

	err := c.HandleRecord(context.Background(), r)
	assert.NoError(t, err)
}

func TestHintConsumer_Handle_ChannelFormat(t *testing.T) {
	pub := centrifugomocks.NewMockPublisher(t)
	pub.EXPECT().
		Publish(mock.Anything, mock.MatchedBy(func(ch string) bool {
			return ch == "canvas:my-session"
		}), mock.Anything).
		Return(nil)

	c := kafka.NewTestHintConsumer(pub, zap.NewNop())
	err := c.HandleRecord(context.Background(), makeHintRecord(t, "my-session"))
	assert.NoError(t, err)
}

func TestHintConsumer_Handle_PassesOriginalBytesToPublish(t *testing.T) {
	record := makeHintRecord(t, "sess-bytes")
	originalValue := record.Value

	pub := centrifugomocks.NewMockPublisher(t)
	pub.EXPECT().
		Publish(mock.Anything, mock.Anything, mock.MatchedBy(func(data []byte) bool {
			return string(data) == string(originalValue)
		})).
		Return(nil)

	c := kafka.NewTestHintConsumer(pub, zap.NewNop())
	err := c.HandleRecord(context.Background(), record)
	assert.NoError(t, err)
}

func TestHintConsumer_Handle_InvalidProtobuf(t *testing.T) {
	pub := centrifugomocks.NewMockPublisher(t)

	c := kafka.NewTestHintConsumer(pub, zap.NewNop())
	r := &kgo.Record{Topic: "canvas.hints", Value: []byte("not valid protobuf !!!")}

	err := c.HandleRecord(context.Background(), r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestHintConsumer_Handle_PublishError(t *testing.T) {
	pub := centrifugomocks.NewMockPublisher(t)
	pub.EXPECT().
		Publish(mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("centrifugo down"))

	c := kafka.NewTestHintConsumer(pub, zap.NewNop())
	err := c.HandleRecord(context.Background(), makeHintRecord(t, "sess-err"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "centrifugo publish hint")
}
