package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

const (
	// RequestIDHeader is the HTTP header for request correlation.
	RequestIDHeader = "X-Request-ID"

	// RequestIDKey is the Gin context key for the request ID.
	RequestIDKey = "request_id"
)

// RequestID injects a unique request ID into every request.
// If the client sends X-Request-ID, it is reused. Otherwise, the OTel TraceID
// from the active span is used (providing log↔trace correlation for free).
// Falls back to a new UUID when no trace context exists.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			span := trace.SpanFromContext(c.Request.Context())
			if sc := span.SpanContext(); sc.HasTraceID() {
				id = sc.TraceID().String()
			} else {
				if v7, err := uuid.NewV7(); err == nil {
					id = v7.String()
				} else {
					id = uuid.New().String()
				}
			}
		}
		c.Set(RequestIDKey, id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}
