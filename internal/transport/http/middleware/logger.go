package middleware

import (
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ZapLogger logs one line per request: what was asked, what was answered and
// how long it took. Successful probes are skipped; anything that failed is
// always logged.
//
// What was asked is the route the request matched and its method, each from a
// set this service wrote down, and never the path, the query or the verb as a
// caller wrote them: those are the caller's own words, and a log line carries
// none.
func ZapLogger(logger *zap.Logger, routes *Routes) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// Read before the handlers run, both together: what was asked is a
		// fact about the request, and by the time the answer exists the
		// request may have been replaced by something downstream. The route
		// is the pattern the framework matched before the first handler ran,
		// named from the router's own table.
		verb := method(c)
		matched := routes.of(c)

		c.Next()

		// A panic is always written, and as an error, whatever status the answer
		// had got to: a handler can panic after it began a 200, and a probe can
		// panic too.
		status := c.Writer.Status()
		panicFields, panicked := panicOf(c)
		if isProbe(matched) && status < 400 && !panicked {
			return
		}

		// A handler that wrote no body leaves the size at -1, which reads as
		// nonsense in a log line; an answer with no body has a body of zero.
		bodySize := max(c.Writer.Size(), 0)

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", verb),
			zap.String("route", matched),
			zap.Duration("duration", time.Since(start)),
			zap.Int("body_size", bodySize),
			zap.String("request_id", RequestIDFrom(c)),
		}
		if errs := c.Errors.ByType(gin.ErrorTypePrivate); len(errs) > 0 {
			fields = append(fields, errorsField(errs))
		}
		fields = append(fields, panicFields...)

		switch {
		case panicked || status >= 500:
			logger.Error("http_request", fields...)
		case status >= 400:
			logger.Warn("http_request", fields...)
		default:
			logger.Info("http_request", fields...)
		}
	}
}

// errorsField says what the errors a request collected were, and not what they
// said. The text of an error is whatever the code that made it had in hand: a
// write that failed names both ends of the connection, the address of whoever
// was at the other end among them. A fault of the system is given in the
// system's own words, which name the fault and nothing else, and anything else
// by its type — the same rule a panic is written by.
func errorsField(errs []*gin.Error) zap.Field {
	kinds := make([]string, len(errs))
	for i, collected := range errs {
		var fault syscall.Errno
		if errors.As(collected.Err, &fault) {
			kinds[i] = fault.Error()
		} else {
			kinds[i] = fmt.Sprintf("an error of type %T", collected.Err)
		}
	}
	return zap.Strings("errors", kinds)
}

// panicOf is what the request panicked with, as the recovery wrote it down,
// and whether it panicked at all.
func panicOf(c *gin.Context) ([]zap.Field, bool) {
	recorded, found := c.Get(panicKey{})
	if !found {
		return nil, false
	}
	fields, isFields := recorded.([]zap.Field)
	return fields, isFields
}
