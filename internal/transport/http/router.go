// Package httpserver builds the HTTP surface of the service.
package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MathTrail/canvas-api/internal/apierror"
	"github.com/MathTrail/canvas-api/internal/transport/http/middleware"
)

// NewRouter builds the service's router: the middleware every request passes
// through, in the order each one needs the others, and the routes it serves.
//
// The order is the point. The last resort comes first, so that a panic in any
// middleware below it is still answered and written down. The security headers
// come next, so that every answer carries them, a refusal and the answer to a
// panic included. The request id comes before the log, so that the request's
// line carries it, and the recovery comes last, next to the handlers, so that
// the request's own line says what it panicked with. What decides who may ask
// at all — the origins a browser may call from, then the access key — joins
// the chain after the recovery, with the first routes that need it.
func NewRouter(log *zap.Logger) *gin.Engine {
	router := gin.New()

	// A path asked with a method it does not take is told so, with the methods
	// it does take, rather than that it does not exist.
	router.HandleMethodNotAllowed = true

	// A path with a slash too many is a path nobody declared. A redirect would
	// answer it with a second request the client never meant to make.
	router.RedirectTrailingSlash = false

	router.Use(middleware.LastResort(log))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestID())
	router.Use(middleware.ZapLogger(log))
	router.Use(middleware.ZapRecovery())

	router.NoRoute(notFound)
	router.NoMethod(methodNotAllowed)

	// /health rather than /healthz: Cloud Run's front end intercepts paths
	// that end in z. HEAD is answered wherever GET is, since a probe may ask
	// either way.
	router.GET("/health", health)
	router.HEAD("/health", health)

	return router
}

// notFound answers a path the service does not serve.
func notFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, apierror.Response{
		Code:    apierror.CodeNotFound,
		Message: "no such endpoint",
	})
}

// methodNotAllowed answers a path asked with a method it does not take. The
// framework has already put the methods it does take in the Allow header.
func methodNotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, apierror.Response{
		Code:    apierror.CodeMethodNotAllowed,
		Message: "this endpoint does not accept that method",
	})
}
