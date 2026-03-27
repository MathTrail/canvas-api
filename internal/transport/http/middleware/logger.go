package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// sensitiveKeys is the deny-list of query parameter names whose values are
// masked in request logs. All comparisons are case-insensitive.
var sensitiveKeys = map[string]struct{}{
	"token":         {},
	"api_key":       {},
	"password":      {},
	"secret":        {},
	"authorization": {},
	"key":           {},
}

// ZapLogger returns a Gin middleware that logs every HTTP request using zap.
// Each log line includes method, path, sanitized query, status, latency,
// client IP, body size, and request ID.
func ZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		status := c.Writer.Status()

		// Skip logging for successful health probes and internal sidecar calls.
		if status < 400 && isInternalPath(path) {
			return
		}

		latency := time.Since(start)

		fields := make([]zap.Field, 0, 10)
		fields = append(fields,
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", sanitizeQuery(query)),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", c.Writer.Size()),
		)

		if id, ok := c.Get(RequestIDKey); ok {
			fields = append(fields, zap.String("request_id", id.(string)))
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		switch {
		case status >= 500:
			logger.Error("request", fields...)
		case status >= 400:
			logger.Warn("request", fields...)
		default:
			logger.Info("request", fields...)
		}
	}
}

// internalPrefixes lists path prefixes that are only logged on errors.
var internalPrefixes = []string{"/health/", "/metrics"}

// isInternalPath reports whether the path matches a probe or sidecar prefix.
func isInternalPath(path string) bool {
	for _, p := range internalPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// sanitizeQuery masks values of sensitive query parameters in the raw query string.
// Unknown keys pass through unchanged. Returns an empty string for empty input.
func sanitizeQuery(raw string) string {
	if raw == "" {
		return ""
	}
	params, err := url.ParseQuery(raw)
	if err != nil {
		return "***"
	}
	for key := range params {
		if _, ok := sensitiveKeys[strings.ToLower(key)]; ok {
			params.Set(key, "***")
		}
	}
	return params.Encode()
}
