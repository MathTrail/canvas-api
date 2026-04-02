package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/mathtrail/canvas-api/internal/apierror"
	"github.com/mathtrail/canvas-api/internal/transport/http/middleware"
)

type tokenResponse struct {
	Token        string `json:"token"`
	Channel      string `json:"channel"`
	ChannelToken string `json:"channel_token"`
}

// TokenHandler handles GET /api/canvas/token.
// It generates Centrifugo connection and channel subscription JWTs for the
// authenticated user and returns them to the canvas UI.
type TokenHandler struct {
	hmacKey []byte
	logger  *zap.Logger
}

func NewTokenHandler(hmacKey string, logger *zap.Logger) *TokenHandler {
	return &TokenHandler{hmacKey: []byte(hmacKey), logger: logger}
}

// Handle generates a Centrifugo connection JWT and a per-channel subscription
// token for the authenticated user. The UI passes these to centrifuge-js.
//
// GET /api/canvas/token?session_id={sessionId}
func (h *TokenHandler) Handle(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" {
		apierror.Abort(c, http.StatusBadRequest, "INVALID_REQUEST", "session_id is required")
		return
	}

	session := middleware.SessionFromContext(c)
	userID := session.Identity.ID
	exp := time.Now().Add(time.Hour)

	// Connection token: identifies the user to Centrifugo.
	connToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"exp": exp.Unix(),
	}).SignedString(h.hmacKey)
	if err != nil {
		h.logger.Error("failed to sign connection token", zap.Error(err))
		apierror.Abort(c, http.StatusInternalServerError, "INTERNAL_ERROR", "token generation failed")
		return
	}

	// Channel subscription token: authorizes the user to subscribe to their
	// private canvas channel. Protected channels require this token.
	channel := "canvas:" + sessionID
	chanToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":     userID,
		"channel": channel,
		"exp":     exp.Unix(),
	}).SignedString(h.hmacKey)
	if err != nil {
		h.logger.Error("failed to sign channel token", zap.Error(err))
		apierror.Abort(c, http.StatusInternalServerError, "INTERNAL_ERROR", "token generation failed")
		return
	}

	c.JSON(http.StatusOK, tokenResponse{
		Token:        connToken,
		Channel:      channel,
		ChannelToken: chanToken,
	})
}
