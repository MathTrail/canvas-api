package httpserver_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap/zaptest"

	"github.com/MathTrail/canvas-api/internal/apierror"
	httpserver "github.com/MathTrail/canvas-api/internal/transport/http"
	"github.com/MathTrail/canvas-api/internal/transport/http/middleware"
	"github.com/MathTrail/canvas-api/internal/version"
)

// The framework keeps its mode in a package variable. A test binary sets it
// once, here, rather than letting a constructor decide it for every other test
// running beside it.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// The probe names the build that answers it, and nothing on the way may keep
// the answer: the delivery reads the commit to know which revision serves, in
// exactly the shape it checks.
func TestHealthNamesTheBuild(t *testing.T) {
	stamp(t)

	rec := call(t, http.MethodGet, "/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want %q", got, "no-store")
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("the body is not a JSON object of strings: %v; body = %s", err, rec.Body.String())
	}
	want := map[string]string{"status": "ok", "version": "v1.2.3", "commit": "abc1234"}
	if !maps.Equal(body, want) {
		t.Errorf("body = %v, want %v", body, want)
	}
}

// The service answers its health probe and nothing else. HEAD is answered
// wherever GET is, since a probe may ask either way; /healthz is not the
// probe, because Cloud Run's front end keeps that path for itself.
func TestRoutes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "the health probe", method: http.MethodGet, path: "/health", want: http.StatusOK},
		{name: "the health probe asked with HEAD", method: http.MethodHead, path: "/health", want: http.StatusOK},
		{name: "the path Cloud Run intercepts", method: http.MethodGet, path: "/healthz", want: http.StatusNotFound},
		{name: "a slash too many", method: http.MethodGet, path: "/health/", want: http.StatusNotFound},
		{name: "an unknown path", method: http.MethodGet, path: "/unknown", want: http.StatusNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if rec := call(t, tc.method, tc.path); rec.Code != tc.want {
				t.Errorf("%s %s status = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
			}
		})
	}
}

// A refusal is the flat error body, with a code a client can branch on and a
// sentence it can relay.
func TestRefusals(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{name: "an unknown path", method: http.MethodGet, path: "/nope", wantStatus: http.StatusNotFound, wantCode: apierror.CodeNotFound},
		{name: "a method the path does not take", method: http.MethodPost, path: "/health", wantStatus: http.StatusMethodNotAllowed, wantCode: apierror.CodeMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rec := call(t, tc.method, tc.path)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			var body apierror.Response
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("body is not an apierror.Response: %v; body = %s", err, rec.Body.String())
			}
			if body.Code != tc.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tc.wantCode)
			}
			if body.Message == "" {
				t.Error("message is empty, want a sentence that can be relayed")
			}
		})
	}
}

// A 405 says which methods the path does take.
func TestA405SaysWhichMethodsWork(t *testing.T) {
	t.Parallel()

	rec := call(t, http.MethodPost, "/health")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got, want := rec.Header().Get("Allow"), "GET, HEAD"; got != want {
		t.Errorf("Allow = %q, want %q", got, want)
	}
}

// Every answer carries a request id and the security headers, a refusal
// included: the chain runs before the router decides there is nothing to
// serve.
func TestEveryAnswerCarriesItsIDAndTheSecurityHeaders(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/health"},
		{method: http.MethodGet, path: "/nope"},
		{method: http.MethodPost, path: "/health"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			rec := call(t, tc.method, tc.path)

			if rec.Header().Get(middleware.RequestIDHeader) == "" {
				t.Errorf("%s is empty, want an id on every answer", middleware.RequestIDHeader)
			}
			for header, want := range map[string]string{
				"Strict-Transport-Security": "max-age=31536000",
				"X-Content-Type-Options":    "nosniff",
				"Referrer-Policy":           "no-referrer",
			} {
				if got := rec.Header().Get(header); got != want {
					t.Errorf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}

func TestAClientsRequestIDIsKept(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)
	req.Header.Set(middleware.RequestIDHeader, "from-the-caller")
	rec := httptest.NewRecorder()
	httpserver.NewRouter(zaptest.NewLogger(t)).ServeHTTP(rec, req)

	if got := rec.Header().Get(middleware.RequestIDHeader); got != "from-the-caller" {
		t.Errorf("%s = %q, want the id the client sent", middleware.RequestIDHeader, got)
	}
}

// call asks a fresh router for method and path.
func call(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, http.NoBody)
	rec := httptest.NewRecorder()
	httpserver.NewRouter(zaptest.NewLogger(t)).ServeHTTP(rec, req)
	return rec
}

// stamp sets the build identity as the linker would, and puts the placeholders
// back when the test ends. The identity is the whole process's, so a test that
// stamps it does not run in parallel: the parallel tests wait until it is over.
func stamp(t *testing.T) {
	t.Helper()

	was := [3]string{version.Version, version.Commit, version.Date}
	version.Version, version.Commit, version.Date = "v1.2.3", "abc1234", "2026-10-12T00:00:00Z"
	t.Cleanup(func() { version.Version, version.Commit, version.Date = was[0], was[1], was[2] })
}
