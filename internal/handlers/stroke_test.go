package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	canvasv1 "github.com/mathtrail/contracts/gen/go/canvas/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/mathtrail/canvas-api/internal/handlers"
	kafkamocks "github.com/mathtrail/canvas-api/internal/kafka/mocks"
	"github.com/mathtrail/canvas-api/internal/ory"
	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testUserID = "user-123"
const testSessionID = "sess-abc"

func validStrokeEvent(t *testing.T) []byte {
	t.Helper()
	event := &canvasv1.CanvasStrokeEvent{
		SessionId: testSessionID,
	}
	b, err := proto.Marshal(event)
	require.NoError(t, err)
	return b
}

// setupStrokeRouter wires a StrokeHandler into a minimal Gin engine with the
// session pre-populated in context (simulating the auth middleware).
func setupStrokeRouter(producer *kafkamocks.MockStrokePublisher) *gin.Engine {
	h := handlers.NewStrokeHandler(producer, zap.NewNop())
	r := gin.New()
	r.POST("/strokes", func(c *gin.Context) {
		c.Set("session", &ory.Session{Active: true, Identity: struct {
			ID string `json:"id"`
		}{ID: testUserID}})
		h.Handle(c)
	})
	return r
}

func TestStrokeHandler_ValidEvent(t *testing.T) {
	producer := kafkamocks.NewMockStrokePublisher(t)
	producer.EXPECT().Publish(mock.Anything, testSessionID, mock.Anything).Return(nil)

	r := setupStrokeRouter(producer)
	body := validStrokeEvent(t)
	req := httptest.NewRequest(http.MethodPost, "/strokes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestStrokeHandler_UserIDStampedFromSession(t *testing.T) {
	producer := kafkamocks.NewMockStrokePublisher(t)
	producer.EXPECT().
		Publish(mock.Anything, testSessionID, mock.MatchedBy(func(data []byte) bool {
			var ev canvasv1.CanvasStrokeEvent
			if err := proto.Unmarshal(data, &ev); err != nil {
				return false
			}
			return ev.UserId == testUserID
		})).
		Return(nil)

	r := setupStrokeRouter(producer)
	body := validStrokeEvent(t)
	req := httptest.NewRequest(http.MethodPost, "/strokes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestStrokeHandler_EmptyBody(t *testing.T) {
	producer := kafkamocks.NewMockStrokePublisher(t)

	r := setupStrokeRouter(producer)
	req := httptest.NewRequest(http.MethodPost, "/strokes", bytes.NewReader([]byte{}))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStrokeHandler_InvalidProtobuf(t *testing.T) {
	producer := kafkamocks.NewMockStrokePublisher(t)

	r := setupStrokeRouter(producer)
	req := httptest.NewRequest(http.MethodPost, "/strokes", bytes.NewReader([]byte("not protobuf")))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStrokeHandler_MissingSessionID(t *testing.T) {
	producer := kafkamocks.NewMockStrokePublisher(t)

	event := &canvasv1.CanvasStrokeEvent{SessionId: ""}
	body, err := proto.Marshal(event)
	require.NoError(t, err)

	r := setupStrokeRouter(producer)
	req := httptest.NewRequest(http.MethodPost, "/strokes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStrokeHandler_PublishError(t *testing.T) {
	producer := kafkamocks.NewMockStrokePublisher(t)
	producer.EXPECT().Publish(mock.Anything, testSessionID, mock.Anything).Return(errors.New("kafka unavailable"))

	r := setupStrokeRouter(producer)
	body := validStrokeEvent(t)
	req := httptest.NewRequest(http.MethodPost, "/strokes", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// Verify that setupStrokeRouter compiles — middleware.SessionFromContext is used
// in the handler and the session key must match.
var _ = middleware.SessionFromContext
