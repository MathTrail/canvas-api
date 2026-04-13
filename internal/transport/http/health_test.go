package httpserver_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	centrifugomocks "github.com/mathtrail/canvas-api/internal/infra/centrifugo/mocks"
	httpserver "github.com/mathtrail/canvas-api/internal/transport/http"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupHealthRouter(h *httpserver.HealthHandler) *gin.Engine {
	r := gin.New()
	r.GET("/health/startup", h.Startup)
	r.GET("/health/live", h.Live)
	r.GET("/health/ready", h.Ready)
	return r
}

func TestHealthHandler_Startup(t *testing.T) {
	pinger := centrifugomocks.NewMockPinger(t)
	r := setupHealthRouter(httpserver.NewHealthHandler(pinger))

	req := httptest.NewRequest(http.MethodGet, "/health/startup", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "started")
}

func TestHealthHandler_Live(t *testing.T) {
	pinger := centrifugomocks.NewMockPinger(t)
	r := setupHealthRouter(httpserver.NewHealthHandler(pinger))

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestHealthHandler_Ready_CentrifugoOK(t *testing.T) {
	pinger := centrifugomocks.NewMockPinger(t)
	pinger.EXPECT().Ping(mock.Anything).Return(nil)
	r := setupHealthRouter(httpserver.NewHealthHandler(pinger))

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

func TestHealthHandler_Ready_CentrifugoDown(t *testing.T) {
	pinger := centrifugomocks.NewMockPinger(t)
	pinger.EXPECT().Ping(mock.Anything).Return(errors.New("connection refused"))
	r := setupHealthRouter(httpserver.NewHealthHandler(pinger))

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "not ready")
	assert.Contains(t, w.Body.String(), "connection refused")
}
