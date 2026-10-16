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

// maxRequestIDLength bounds what a client may dictate: an id is a correlation
// handle, and an unbounded one is a way to write anything into a log line.
const maxRequestIDLength = 64

// RequestID gives every request an id and echoes it back. A client that sends
// a usable one keeps it, so that a trace through somebody else's logs and a
// trace through ours use the same word; anything else is replaced.
//
// The id is kept in the request's context, the one place both the handlers of
// this framework and code that knows nothing of it can read it from, so that
// every line of the request carries one id.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !usableRequestID(id) {
			id = newRequestID()
		}
		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), id))
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// usableRequestID reports whether a client's id can be carried as it stands.
//
// The alphabet is deliberately narrow: letters, digits and the four separators
// every correlation format in the wild is built from — a UUID, a hex trace id
// and a W3C traceparent all fit, and nothing else needs to. What it keeps out
// is a value that means something to a reader rather than to a machine: a
// newline, a control byte, a quote. Neither our JSON log nor the header writer
// would be fooled by one today, but the id also travels into whatever we
// correlate with next, and this is the one place that can be said once.
func usableRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, b := range []byte(id) {
		switch {
		case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		case b == '-', b == '_', b == '.', b == ':':
		default:
			return false
		}
	}
	return true
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
