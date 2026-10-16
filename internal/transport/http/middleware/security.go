package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders puts on every answer what tells a browser how to treat it:
// reach this host over HTTPS only, for a year; believe the type an answer
// declares rather than guess another; and send no address of ours onward in a
// referrer.
//
// They are set before anything else runs, so that every answer carries them —
// a refusal and an answer to a panic included.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("Strict-Transport-Security", "max-age=31536000")
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}
