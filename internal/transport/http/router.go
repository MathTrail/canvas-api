package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/config"
	"github.com/mathtrail/canvas-api/internal/handlers"
	"github.com/mathtrail/canvas-api/internal/ory"
	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

// NewRouter creates and configures the Gin router with all routes and middleware.
func NewRouter(
	strokeHandler *handlers.StrokeHandler,
	tokenHandler *handlers.TokenHandler,
	healthHandler *HealthHandler,
	cfg *config.Config,
	logger *zap.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware.
	// Order matters: otelgin wraps everything for tracing, ZapRecovery catches
	// panics from all downstream middleware and handlers.
	router.Use(otelgin.Middleware("canvas-api"))    // extracts W3C traceparent, creates child spans
	router.Use(middleware.ZapRecovery(logger))       // must be early to catch panics in middleware below
	router.Use(middleware.RequestID())               // links to OTel TraceID when X-Request-ID is absent
	router.Use(middleware.ZapLogger(logger))
	router.Use(middleware.CORS(cfg.AllowedOrigins)) // must be after logging so CORS headers appear in logs

	// Observability
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoints (for Kubernetes probes)
	router.GET("/health/startup", healthHandler.Startup)
	router.GET("/health/liveness", healthHandler.Live)
	router.GET("/health/ready", healthHandler.Ready)

	// API routes — protected by Ory Kratos session auth
	api := router.Group("/api/canvas")
	api.Use(middleware.Auth(ory.NewClient(cfg.OryKratosURL)))
	{
		api.GET("/token", tokenHandler.Handle)
		api.POST("/strokes", strokeHandler.Handle)
	}

	return router
}
