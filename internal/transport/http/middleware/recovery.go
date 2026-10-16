package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MathTrail/canvas-api/internal/apierror"
	"github.com/MathTrail/canvas-api/internal/logger"
)

// panicKey holds, for the length of one request, what the request panicked
// with and where, so that the request's own line in the log says it. It is a
// type of its own, so that nothing else a request carries can be taken for it.
type panicKey struct{}

// ZapRecovery turns a panic of a handler into a plain JSON answer. One request
// must not take the process down: another child is in the middle of a task on
// the same instance.
//
// It stands next to the handlers and writes no line of its own. What the
// request panicked with, and where, goes into the request's line, which the
// request log writes when it sees the answer: one fault, one line, and the
// panicked request logged like any other.
//
// A handler that panics with http.ErrAbortHandler, which is how net/http is
// told to end a response unfinished, is passed on as it asked: the server
// then drops the request silently, and nothing above takes it for an answer.
func ZapRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		recovering(c, func(recovered any) {
			c.Set(panicKey{}, []zap.Field{logger.PanicValue(recovered), logger.PanicStack()})
		})
	}
}

// LastResort catches what ZapRecovery cannot: a panic in the middleware above
// it, whose request never reaches the request log. It writes the line itself,
// naming the request as the request log does, and answers and passes an abort
// on as ZapRecovery does.
func LastResort(log *zap.Logger, routes *Routes) gin.HandlerFunc {
	return func(c *gin.Context) {
		recovering(c, func(recovered any) {
			log.Error("panic",
				logger.PanicValue(recovered),
				logger.PanicStack(),
				zap.String("method", method(c)),
				zap.String("route", routes.of(c)),
				zap.String("request_id", RequestIDFrom(c)),
			)
		})
	}
}

// recovering runs the handlers after it and catches a panic of theirs: an
// abort is passed on to the server, and anything else is written down by
// record and answered.
func recovering(c *gin.Context, record func(recovered any)) {
	defer func() {
		recovered := recover()
		switch {
		case recovered == nil:
			return
		case aborted(recovered):
			panic(http.ErrAbortHandler)
		}
		record(recovered)
		answerPanic(c)
	}()
	c.Next()
}

// aborted says whether a panic is a handler asking net/http to end its
// response unfinished, however it was wrapped on the way. It is passed on bare,
// because that is the one value the server knows to drop without a word.
func aborted(recovered any) bool {
	err, isError := recovered.(error)
	return isError && errors.Is(err, http.ErrAbortHandler)
}

// everyAnswerCarries are the headers the middleware above the handlers puts on
// every answer, an answer to a panic included.
var everyAnswerCarries = map[string]struct{}{
	"Strict-Transport-Security": {},
	"X-Content-Type-Options":    {},
	"Referrer-Policy":           {},
	RequestIDHeader:             {},
}

// answerPanic answers a request that panicked. A handler that panicked halfway
// through its answer has already sent a status and headers, and writing a
// second one corrupts what the client is reading: all that is left then is to
// stop.
//
// What the handler had already put in the headers is dropped first: a type, a
// length or a caching rule it set for the answer it meant to give would
// otherwise go out with the error in its place. What the chain above put there
// for every answer stays.
func answerPanic(c *gin.Context) {
	if c.Writer.Written() {
		c.Abort()
		return
	}
	header := c.Writer.Header()
	for name := range header {
		if _, kept := everyAnswerCarries[name]; !kept {
			header.Del(name)
		}
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.Response{
		Code:    apierror.CodeInternal,
		Message: "an unexpected error occurred",
	})
}
