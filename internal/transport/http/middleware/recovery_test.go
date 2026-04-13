package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

func TestZapRecovery_PanicReturns500(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ZapRecovery(zap.NewNop()))
	r.GET("/boom", func(c *gin.Context) {
		panic("something went horribly wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body["code"])
	assert.NotEmpty(t, body["message"])
}

func TestZapRecovery_NoPanic_PassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ZapRecovery(zap.NewNop()))
	r.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestZapRecovery_PanicWithErrorValue(t *testing.T) {
	r := gin.New()
	r.Use(middleware.ZapRecovery(zap.NewNop()))
	r.GET("/err", func(c *gin.Context) {
		// Panic with a non-sentinel error — recovery must catch it.
		panic(assert.AnError)
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
