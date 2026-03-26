package middleware

import (
	"context"
	"net/http"

	"github.com/mathtrail/canvas-api/internal/ory"
)

type contextKey string

const sessionKey contextKey = "session"

// Auth validates the Ory Kratos session cookie and stores the session in the
// request context. Returns 401 if the session is missing or invalid.
func Auth(kratosURL string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := ory.WhoAmI(r.Context(), kratosURL, r.Cookies())
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), sessionKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SessionFromContext retrieves the validated Ory session from the context.
// Panics if called outside an Auth-protected route.
func SessionFromContext(ctx context.Context) *ory.Session {
	return ctx.Value(sessionKey).(*ory.Session)
}
