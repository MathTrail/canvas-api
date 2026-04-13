package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

func newRequestIDRouter() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		id, _ := c.Get(middleware.RequestIDKey)
		c.JSON(http.StatusOK, gin.H{"id": id})
	})
	return r
}

func TestRequestID_ClientHeaderReused(t *testing.T) {
	r := newRequestIDRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, "my-request-id")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, "my-request-id", w.Header().Get(middleware.RequestIDHeader))
	assert.Contains(t, w.Body.String(), "my-request-id")
}

func TestRequestID_GeneratedWhenAbsent(t *testing.T) {
	r := newRequestIDRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	id := w.Header().Get(middleware.RequestIDHeader)
	assert.NotEmpty(t, id)
	assert.Contains(t, w.Body.String(), id)
}

func TestRequestID_TooLongHeaderIgnored(t *testing.T) {
	r := newRequestIDRouter()
	longID := strings.Repeat("x", 65)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, longID)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Long header is discarded; a generated ID is used instead.
	got := w.Header().Get(middleware.RequestIDHeader)
	require.NotEmpty(t, got)
	assert.NotEqual(t, longID, got)
}

func TestRequestID_Exactly64CharsAccepted(t *testing.T) {
	r := newRequestIDRouter()
	id64 := strings.Repeat("a", 64)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, id64)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, id64, w.Header().Get(middleware.RequestIDHeader))
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	r := newRequestIDRouter()
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		id := w.Header().Get(middleware.RequestIDHeader)
		assert.NotEmpty(t, id)
		ids[id] = true
	}

	assert.Len(t, ids, 10, "each request should get a unique ID")
}

func TestRequestID_StoredInContext(t *testing.T) {
	var ctxID string
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		v, ok := c.Get(middleware.RequestIDKey)
		require.True(t, ok)
		ctxID = v.(string)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.RequestIDHeader, "ctx-test-id")
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "ctx-test-id", ctxID)
}
