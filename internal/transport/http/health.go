package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mathtrail/canvas-api/internal/infra/centrifugo"
)

// HealthHandler serves Kubernetes health probe endpoints.
type HealthHandler struct {
	centrifugo *centrifugo.Client
}

// NewHealthHandler creates a HealthHandler backed by the given Centrifugo client.
func NewHealthHandler(c *centrifugo.Client) *HealthHandler {
	return &HealthHandler{centrifugo: c}
}

// Startup indicates that the application has started.
func (h *HealthHandler) Startup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// Live indicates that the application is running.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready verifies Centrifugo connectivity before reporting ready.
func (h *HealthHandler) Ready(c *gin.Context) {
	if err := h.centrifugo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "centrifugo: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
