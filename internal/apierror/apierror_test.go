package apierror_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mathtrail/canvas-api/internal/apierror"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAbort(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		code       string
		message    string
		wantStatus int
		wantBody   apierror.Response
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			code:       "INVALID_REQUEST",
			message:    "session_id is required",
			wantStatus: http.StatusBadRequest,
			wantBody:   apierror.Response{Code: "INVALID_REQUEST", Message: "session_id is required"},
		},
		{
			name:       "unauthorized",
			status:     http.StatusUnauthorized,
			code:       "UNAUTHORIZED",
			message:    "unauthorized",
			wantStatus: http.StatusUnauthorized,
			wantBody:   apierror.Response{Code: "UNAUTHORIZED", Message: "unauthorized"},
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			code:       "INTERNAL_ERROR",
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
			wantBody:   apierror.Response{Code: "INTERNAL_ERROR", Message: "something went wrong"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, engine := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

			called := false
			engine.GET("/", func(c *gin.Context) {
				apierror.Abort(c, tc.status, tc.code, tc.message)
				called = true // Abort does not stop the current handler
			}, func(c *gin.Context) {
				t.Error("next handler must not be called after Abort")
			})

			engine.ServeHTTP(w, c.Request)

			assert.True(t, called)
			assert.Equal(t, tc.wantStatus, w.Code)

			var got apierror.Response
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
			assert.Equal(t, tc.wantBody, got)
		})
	}
}
