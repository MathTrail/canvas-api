package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mathtrail/canvas-api/internal/apierror"
	"github.com/mathtrail/canvas-api/internal/ory"
)

const sessionKey = "session"

// Auth validates the Ory Kratos session cookie and stores the session in the
// Gin context. Aborts with 401 if the session is missing or invalid.
func Auth(client *ory.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := client.WhoAmI(c.Request.Context(), c.Request.Cookies())
		if err != nil {
			apierror.Abort(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
			return
		}
		c.Set(sessionKey, session)
		c.Next()
	}
}

// SessionFromContext retrieves the validated Ory session from the Gin context.
// Panics if called outside an Auth-protected route.
func SessionFromContext(c *gin.Context) *ory.Session {
	return c.MustGet(sessionKey).(*ory.Session)
}
