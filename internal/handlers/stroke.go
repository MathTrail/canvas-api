package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	canvasv1 "github.com/mathtrail/contracts/gen/go/canvas/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/mathtrail/canvas-api/internal/apierror"
	"github.com/mathtrail/canvas-api/internal/kafka"
	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

// StrokeHandler handles POST /api/canvas/strokes.
// It validates the Protobuf payload, stamps the user ID from the authenticated
// session, and publishes the event to AutoMQ.
type StrokeHandler struct {
	producer *kafka.Producer
	logger   *zap.Logger
}

func NewStrokeHandler(producer *kafka.Producer, logger *zap.Logger) *StrokeHandler {
	return &StrokeHandler{producer: producer, logger: logger}
}

// Handle receives a Protobuf-encoded CanvasStrokeEvent from the canvas UI,
// validates it, and publishes it to AutoMQ.
//
// POST /api/canvas/strokes
// Content-Type: application/octet-stream
// Body: Protobuf-encoded CanvasStrokeEvent
func (h *StrokeHandler) Handle(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1 MB limit

	body, err := c.GetRawData()
	if err != nil {
		h.logger.Warn("failed to read stroke body", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.Response{
			Code:    "INVALID_REQUEST",
			Message: "failed to read request body",
		})
		return
	}

	var event canvasv1.CanvasStrokeEvent
	if err := proto.Unmarshal(body, &event); err != nil {
		h.logger.Warn("invalid protobuf payload", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.Response{
			Code:    "INVALID_REQUEST",
			Message: "invalid protobuf payload",
		})
		return
	}

	// Stamp user_id from the validated session so the client cannot spoof it.
	session := middleware.SessionFromContext(c)
	event.UserId = session.Identity.ID

	data, err := proto.Marshal(&event)
	if err != nil {
		h.logger.Error("failed to marshal stroke event", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.Response{
			Code:    "INTERNAL_ERROR",
			Message: "failed to process stroke event",
		})
		return
	}

	if err := h.producer.Publish(c.Request.Context(), event.SessionId, data); err != nil {
		h.logger.Error("failed to publish stroke event",
			zap.Error(err),
			zap.String("session_id", event.SessionId),
		)
		c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.Response{
			Code:    "INTERNAL_ERROR",
			Message: "failed to publish stroke event",
		})
		return
	}

	c.Status(http.StatusAccepted)
}
