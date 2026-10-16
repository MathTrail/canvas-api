// Package middleware holds the gin middleware every request passes through.
package middleware

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MathTrail/canvas-api/internal/logger"
)

// RequestIDHeader is the header a request is correlated by.
const RequestIDHeader = "X-Request-Id"

// requestsSeen is what keeps two fallback identifiers apart when the source
// of randomness has stopped answering.
var requestsSeen atomic.Uint64

// RequestID gives every request an id of the service's own and sends it back
// in the answer. An id a client sends is not read: no client of this service
// needs one kept, and an id taken from a request would put a caller's words
// into every line of the log.
//
// The id is kept in the request's context, the one place both the handlers of
// this framework and code that knows nothing of it can read it from, so that
// every line of the request carries one id.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := newRequestID()
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// newRequestID prefers a version 7 identifier, whose leading bits are a
// timestamp, so that ids sort by the order the requests arrived.
//
// A second kind of identifier would be no use as a fallback: every version
// asks the same source of randomness, so they fail together, and the one that
// takes no error panics instead of saying so. What follows a failure is the
// clock and a counter — a correlation handle is not a secret, and two lines
// only have to be told apart.
func newRequestID() string {
	if v7, err := uuid.NewV7(); err == nil {
		return v7.String()
	}
	return fallbackRequestID()
}

// fallbackRequestID is what a request is correlated by when there is no
// randomness to be had. The clock puts ids in the order they were handed out,
// and the counter keeps two within the same nanosecond apart.
func fallbackRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" +
		strconv.FormatUint(requestsSeen.Add(1), 36)
}

// RequestIDFrom returns the request id set by RequestID, or an empty string.
func RequestIDFrom(c *gin.Context) string {
	if c.Request == nil {
		return ""
	}
	return logger.RequestID(c.Request.Context())
}
