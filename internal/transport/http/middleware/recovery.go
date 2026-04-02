package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/apierror"
)

// panicStackSkip is the number of zap/runtime frames to drop from the top of
// the captured stacktrace so the trace starts at application code.
const panicStackSkip = 3

// ZapRecovery returns a Gin middleware that recovers from panics and logs them
// with zap. It returns a structured JSON error response so clients can handle
// the failure gracefully.
func ZapRecovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// Mark the active OTel span as failed so the trace appears as an error in Tempo.
		span := trace.SpanFromContext(c.Request.Context())
		span.SetStatus(codes.Error, "panic recovered")
		span.RecordError(fmt.Errorf("%v", recovered))

		logger.Error("panic recovered",
			zap.Any("error", recovered),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.StackSkip("stack", panicStackSkip),
		)
		apierror.Abort(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	})
}
