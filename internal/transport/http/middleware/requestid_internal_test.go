package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/MathTrail/canvas-api/internal/logger"
)

// When the source of randomness stops answering there is still a request to
// correlate. The identifier that follows has to pass the same rule as any
// other, and two of them have to be different.
func TestTheFallbackIdentifierIsUsableAndUnique(t *testing.T) {
	t.Parallel()

	first := fallbackRequestID()
	second := fallbackRequestID()

	for _, id := range []string{first, second} {
		if !usableRequestID(id) {
			t.Errorf("fallbackRequestID() = %q, want an id the middleware would keep", id)
		}
	}
	if first == second {
		t.Errorf("two identifiers came back the same: %q", first)
	}
}

// A request that arrives once the source of randomness has stopped answering
// is still given an identifier: the fallback, under the same rule and just as
// different from the one before it.
func TestAnIdentifierIsMadeWithoutRandomness(t *testing.T) {
	// Not parallel: the source of randomness belongs to the whole process, and
	// it is put back before any test that runs in parallel starts.
	uuid.SetRand(iotest.ErrReader(errors.New("no randomness left")))
	t.Cleanup(func() { uuid.SetRand(nil) })

	first, second := newRequestID(), newRequestID()
	for _, id := range []string{first, second} {
		if !usableRequestID(id) {
			t.Errorf("newRequestID() = %q, want an id the middleware would keep", id)
		}
	}
	if first == second {
		t.Errorf("two requests were given the same id %q, want them told apart", first)
	}
}

// Whatever a client sends as its request id, the id the request is known by
// is one the rule keeps — short, and made only of the bytes a correlation
// format is built from — and a client's id is kept exactly when the rule keeps
// it. The header is the one part of a request the skeleton reads as it was
// written, so this is what a log line and an answer can never be made to carry.
func FuzzRequestID(f *testing.F) {
	for _, seed := range []string{
		"",
		"from-the-caller",
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"line\nbreak",
		`"quoted"`,
		strings.Repeat("a", 64),
		strings.Repeat("a", 65),
		"\x00",
		"ünïcödé",
	} {
		f.Add(seed)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, logger.RequestID(c.Request.Context()))
	})

	f.Fuzz(func(t *testing.T, sent string) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
		req.Header[RequestIDHeader] = []string{sent}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		echoed := rec.Header().Get(RequestIDHeader)
		if !plainIdentifier(echoed) {
			t.Fatalf("echoed id %q for %q, want 1 to 64 letters, digits, '-', '_', '.' or ':'", echoed, sent)
		}
		if logged := rec.Body.String(); logged != echoed {
			t.Fatalf("the context carries %q, the header %q, want one id", logged, echoed)
		}
		if plainIdentifier(sent) != (echoed == sent) {
			t.Fatalf("sent %q, echoed %q: a client's id is kept exactly when the rule keeps it", sent, echoed)
		}
	})
}

// plainIdentifier is the rule a request id is held to, written out here apart
// from the code it checks, so that a rule loosened there is caught rather than
// agreed with.
func plainIdentifier(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.:", r) {
			return false
		}
	}
	return true
}
