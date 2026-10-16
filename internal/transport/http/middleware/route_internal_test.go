package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Every known method is reported as itself, and anything else as other, in any
// case a caller chose to spell it.
func TestAMethodIsReportedFromTheClosedSet(t *testing.T) {
	t.Parallel()

	for sent, want := range map[string]string{
		http.MethodGet:     http.MethodGet,
		http.MethodHead:    http.MethodHead,
		http.MethodPost:    http.MethodPost,
		http.MethodPut:     http.MethodPut,
		http.MethodPatch:   http.MethodPatch,
		http.MethodDelete:  http.MethodDelete,
		http.MethodOptions: http.MethodOptions,
		http.MethodConnect: http.MethodConnect,
		http.MethodTrace:   http.MethodTrace,
		"get":              otherLabel,
		"MASHA":            otherLabel,
	} {
		t.Run(sent, func(t *testing.T) {
			t.Parallel()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			c.Request.Method = sent
			if got := method(c); got != want {
				t.Errorf("method() = %q, want %q", got, want)
			}
		})
	}
}
