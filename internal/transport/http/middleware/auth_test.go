package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/ory"
	orymocks "github.com/mathtrail/canvas-api/internal/ory/mocks"
	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
	_ = zap.NewNop() // ensure zap import is live
}

func newAuthRouter(validator ory.SessionValidator) *gin.Engine {
	r := gin.New()
	r.GET("/protected", middleware.Auth(validator), func(c *gin.Context) {
		s := middleware.SessionFromContext(c)
		c.JSON(http.StatusOK, gin.H{"user_id": s.Identity.ID})
	})
	return r
}

func TestAuth_ValidSession(t *testing.T) {
	sess := &ory.Session{Active: true, Identity: struct {
		ID string `json:"id"`
	}{ID: "user-42"}}

	v := orymocks.NewMockSessionValidator(t)
	v.EXPECT().WhoAmI(mock.Anything, mock.Anything).Return(sess, nil)

	r := newAuthRouter(v)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-42")
}

func TestAuth_InvalidSession(t *testing.T) {
	v := orymocks.NewMockSessionValidator(t)
	v.EXPECT().WhoAmI(mock.Anything, mock.Anything).Return(nil, errors.New("unauthenticated"))

	r := newAuthRouter(v)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_NextNotCalledOnError(t *testing.T) {
	v := orymocks.NewMockSessionValidator(t)
	v.EXPECT().WhoAmI(mock.Anything, mock.Anything).Return(nil, ory.ErrUnauthenticated)

	nextCalled := false
	r := gin.New()
	r.GET("/protected", middleware.Auth(v), func(c *gin.Context) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.False(t, nextCalled)
}

func TestSessionFromContext_ReturnsSession(t *testing.T) {
	sess := &ory.Session{Active: true, Identity: struct {
		ID string `json:"id"`
	}{ID: "ctx-user"}}

	v := orymocks.NewMockSessionValidator(t)
	v.EXPECT().WhoAmI(mock.Anything, mock.Anything).Return(sess, nil)

	var got *ory.Session
	r := gin.New()
	r.GET("/", middleware.Auth(v), func(c *gin.Context) {
		got = middleware.SessionFromContext(c)
		c.Status(http.StatusOK)
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	require.NotNil(t, got)
	assert.Equal(t, "ctx-user", got.Identity.ID)
}
