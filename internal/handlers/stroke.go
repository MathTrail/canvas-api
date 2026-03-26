package handlers

import (
	"io"
	"net/http"

	canvasv1 "github.com/mathtrail/contracts/gen/go/canvas/v1"
	"github.com/mathtrail/canvas-api/internal/kafka"
	"github.com/mathtrail/canvas-api/internal/middleware"
	"google.golang.org/protobuf/proto"
)

// Strokes receives a Protobuf-encoded CanvasStrokeEvent from the canvas UI,
// validates it, and publishes it to AutoMQ.
//
// POST /api/canvas/strokes
// Content-Type: application/octet-stream
// Body: Protobuf-encoded CanvasStrokeEvent
func Strokes(producer *kafka.Producer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}

		var event canvasv1.CanvasStrokeEvent
		if err := proto.Unmarshal(body, &event); err != nil {
			http.Error(w, "invalid protobuf payload", http.StatusBadRequest)
			return
		}

		// Stamp user_id from the validated session so the client cannot spoof it.
		session := middleware.SessionFromContext(r.Context())
		event.UserId = session.Identity.ID

		data, err := proto.Marshal(&event)
		if err != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			return
		}

		if err := producer.Publish(r.Context(), event.SessionId, data); err != nil {
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
