package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// CORS returns a CORS middleware configured for canvas-api.
// allowedOrigins must include the ui-web shell origin and localhost:3001
// (canvas-api/ui dev server). AllowCredentials is required so that Ory
// session cookies are forwarded on cross-origin requests from the MFE.
func CORS(allowedOrigins []string) func(next http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type", "Cookie"},
		AllowCredentials: true,
	})
	return c.Handler
}
