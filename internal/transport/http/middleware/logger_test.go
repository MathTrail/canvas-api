package middleware

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeQuery_Empty(t *testing.T) {
	assert.Equal(t, "", sanitizeQuery(""))
}

func TestSanitizeQuery_NoSensitiveKeys(t *testing.T) {
	raw := "session_id=abc&foo=bar"
	got := sanitizeQuery(raw)
	parsed, _ := url.ParseQuery(got)
	assert.Equal(t, "abc", parsed.Get("session_id"))
	assert.Equal(t, "bar", parsed.Get("foo"))
}

func TestSanitizeQuery_MasksToken(t *testing.T) {
	got := sanitizeQuery("token=supersecret&session_id=abc")
	parsed, _ := url.ParseQuery(got)
	assert.Equal(t, "***", parsed.Get("token"))
	assert.Equal(t, "abc", parsed.Get("session_id"))
}

func TestSanitizeQuery_MasksAPIKey(t *testing.T) {
	got := sanitizeQuery("api_key=mykey&other=val")
	parsed, _ := url.ParseQuery(got)
	assert.Equal(t, "***", parsed.Get("api_key"))
	assert.Equal(t, "val", parsed.Get("other"))
}

func TestSanitizeQuery_MasksPassword(t *testing.T) {
	got := sanitizeQuery("password=hunter2")
	parsed, _ := url.ParseQuery(got)
	assert.Equal(t, "***", parsed.Get("password"))
}

func TestSanitizeQuery_CaseInsensitive(t *testing.T) {
	got := sanitizeQuery("TOKEN=value&Key=other")
	parsed, _ := url.ParseQuery(got)
	assert.Equal(t, "***", parsed.Get("TOKEN"))
	assert.Equal(t, "***", parsed.Get("Key"))
}

func TestSanitizeQuery_InvalidQuery(t *testing.T) {
	// url.ParseQuery fails on malformed percent-encoding — returns "***".
	got := sanitizeQuery("%zz=bad")
	assert.Equal(t, "***", got)
}

func TestIsInternalPath_HealthPrefixes(t *testing.T) {
	for _, path := range []string{"/health/live", "/health/ready", "/health/startup", "/health/"} {
		assert.True(t, isInternalPath(path), path)
	}
}

func TestIsInternalPath_Metrics(t *testing.T) {
	assert.True(t, isInternalPath("/metrics"))
}

func TestIsInternalPath_APIRoutes(t *testing.T) {
	for _, path := range []string{"/api/canvas/strokes", "/api/canvas/token", "/"} {
		assert.False(t, isInternalPath(path), path)
	}
}
