package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mathtrail/canvas-api/internal/middleware"
)

type tokenResponse struct {
	Token        string `json:"token"`
	Channel      string `json:"channel"`
	ChannelToken string `json:"channel_token"`
}

// Token generates a Centrifugo connection JWT and a per-channel subscription
// token for the authenticated user. The UI passes these to centrifuge-js.
//
// GET /api/canvas/token?session_id={sessionId}
func Token(hmacKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("session_id")
		if sessionID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		session := middleware.SessionFromContext(c)
		userID := session.Identity.ID

		key := []byte(hmacKey)
		exp := time.Now().Add(time.Hour)

		// Connection token: identifies the user to Centrifugo.
		connToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": userID,
			"exp": exp.Unix(),
		}).SignedString(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
			return
		}

		// Channel subscription token: authorizes the user to subscribe to their
		// private canvas channel. Protected channels require this token.
		channel := "canvas:" + sessionID
		chanToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":     userID,
			"channel": channel,
			"exp":     exp.Unix(),
		}).SignedString(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
			return
		}

		c.JSON(http.StatusOK, tokenResponse{
			Token:        connToken,
			Channel:      channel,
			ChannelToken: chanToken,
		})
	}
}
