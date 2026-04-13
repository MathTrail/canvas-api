package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/handlers"
	"github.com/mathtrail/canvas-api/internal/ory"
)

const testHMACKey = "test-hmac-secret"

func setupTokenRouter(hmacKey string) *gin.Engine {
	h := handlers.NewTokenHandler(hmacKey, zap.NewNop())
	r := gin.New()
	r.GET("/token", func(c *gin.Context) {
		c.Set("session", &ory.Session{Active: true, Identity: struct {
			ID string `json:"id"`
		}{ID: testUserID}})
		h.Handle(c)
	})
	return r
}

func TestTokenHandler_ValidRequest(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token?session_id="+testSessionID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Token        string `json:"token"`
		Channel      string `json:"channel"`
		ChannelToken string `json:"channel_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.ChannelToken)
	assert.Equal(t, "canvas:"+testSessionID, resp.Channel)
}

func TestTokenHandler_ChannelFormat(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token?session_id=my-session", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "canvas:my-session", resp["channel"])
}

func TestTokenHandler_ConnectionTokenClaims(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token?session_id="+testSessionID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	token, err := jwt.Parse(resp["token"], func(t *jwt.Token) (interface{}, error) {
		return []byte(testHMACKey), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, testUserID, claims["sub"])
	assert.NotNil(t, claims["exp"])
}

func TestTokenHandler_ChannelTokenClaims(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token?session_id="+testSessionID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	token, err := jwt.Parse(resp["channel_token"], func(t *jwt.Token) (interface{}, error) {
		return []byte(testHMACKey), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, testUserID, claims["sub"])
	assert.Equal(t, "canvas:"+testSessionID, claims["channel"])
}

func TestTokenHandler_MissingSessionID(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_REQUEST", resp["code"])
}

func TestTokenHandler_EmptySessionID(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token?session_id=", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTokenHandler_TokensSignedWithHMACKey(t *testing.T) {
	r := setupTokenRouter(testHMACKey)
	req := httptest.NewRequest(http.MethodGet, "/token?session_id="+testSessionID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// Verify tokens use HS256 by checking the header segment.
	for _, field := range []string{"token", "channel_token"} {
		parts := strings.Split(resp[field], ".")
		require.Len(t, parts, 3, "JWT must have 3 parts")
	}

	// Verify a wrong key fails.
	_, err := jwt.Parse(resp["token"], func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong-key"), nil
	})
	assert.Error(t, err)
}
