package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/mathtrail/canvas-api/internal/config"
	centrifugoclient "github.com/mathtrail/canvas-api/internal/infra/centrifugo"
	"github.com/mathtrail/canvas-api/internal/handlers"
	"github.com/mathtrail/canvas-api/internal/kafka"
	"github.com/mathtrail/canvas-api/internal/middleware"
)

// NewRouter creates and configures the Gin engine with all routes and middleware.
func NewRouter(cfg *config.Config, producer *kafka.Producer, cClient *centrifugoclient.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.GET("/health", handlers.Health)

	api := r.Group("/api/canvas")
	api.Use(middleware.Auth(cfg.OryKratosURL))
	{
		api.GET("/token", handlers.Token(cfg.CentrifugoHMACKey))
		api.POST("/strokes", handlers.Strokes(producer))
	}

	return r
}
