package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

func newCORSRouter(origins []string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(origins))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.OPTIONS("/", func(c *gin.Context) {})
	return r
}

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	r := newCORSRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	r := newCORSRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_NoOriginHeader_NoHeaders(t *testing.T) {
	r := newCORSRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_Preflight_Returns204(t *testing.T) {
	r := newCORSRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_Preflight_DisallowedOrigin_NoHeaders(t *testing.T) {
	r := newCORSRouter([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://attacker.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	origins := []string{"https://app.example.com", "http://localhost:3000"}
	r := newCORSRouter(origins)

	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", origin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, origin, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
