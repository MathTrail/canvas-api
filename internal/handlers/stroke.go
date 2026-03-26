package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	canvasv1 "github.com/mathtrail/contracts/gen/go/canvas/v1"
	"google.golang.org/protobuf/proto"

	"github.com/mathtrail/canvas-api/internal/kafka"
	"github.com/mathtrail/canvas-api/internal/middleware"
)

// Strokes receives a Protobuf-encoded CanvasStrokeEvent from the canvas UI,
// validates it, and publishes it to AutoMQ.
//
// POST /api/canvas/strokes
// Content-Type: application/octet-stream
// Body: Protobuf-encoded CanvasStrokeEvent
func Strokes(producer *kafka.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20) // 1 MB limit

		body, err := c.GetRawData()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "read body failed"})
			return
		}

		var event canvasv1.CanvasStrokeEvent
		if err := proto.Unmarshal(body, &event); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid protobuf payload"})
			return
		}

		// Stamp user_id from the validated session so the client cannot spoof it.
		session := middleware.SessionFromContext(c)
		event.UserId = session.Identity.ID

		data, err := proto.Marshal(&event)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "marshal failed"})
			return
		}

		if err := producer.Publish(c.Request.Context(), event.SessionId, data); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "publish failed"})
			return
		}

		c.Status(http.StatusAccepted)
	}
}
