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

// A request is named by a pattern from the table of the router the names were
// made for: a pattern it serves is written as itself, and a pattern no route of
// it serves — even one another router matched — is written as other.
func TestARouteIsNamedFromTheRoutersOwnTable(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	served := gin.New()
	served.GET("/tasks/:id", func(*gin.Context) {})
	stranger := gin.New()

	for _, tc := range []struct {
		name   string
		routes *Routes
		want   string
	}{
		{name: "the router's own pattern", routes: NewRoutes(served), want: "/tasks/:id"},
		{name: "a pattern the table does not hold", routes: NewRoutes(stranger), want: otherLabel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var named string
			engine := gin.New()
			engine.GET("/tasks/:id", func(c *gin.Context) { named = tc.routes.of(c) })
			engine.ServeHTTP(httptest.NewRecorder(),
				httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/tasks/42", http.NoBody))
			if named != tc.want {
				t.Errorf("routes.of() = %q, want %q", named, tc.want)
			}
		})
	}
}
