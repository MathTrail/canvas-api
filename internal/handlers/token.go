package handlers

import (
	"encoding/json"
	"net/http"
	"time"

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
func Token(hmacKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id is required", http.StatusBadRequest)
			return
		}

		session := middleware.SessionFromContext(r.Context())
		userID := session.Identity.ID

		key := []byte(hmacKey)
		now := time.Now()
		exp := now.Add(time.Hour)

		// Connection token: identifies the user to Centrifugo.
		connToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": userID,
			"exp": exp.Unix(),
		}).SignedString(key)
		if err != nil {
			http.Error(w, "token generation failed", http.StatusInternalServerError)
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
			http.Error(w, "token generation failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			Token:        connToken,
			Channel:      channel,
			ChannelToken: chanToken,
		})
	}
}
