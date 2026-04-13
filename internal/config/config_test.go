package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mathtrail/canvas-api/internal/config"
)

// validEnv sets the minimum required env vars for a successful Load.
func validEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AUTOMQ_BROKERS", "broker1:9092,broker2:9092")
	t.Setenv("CENTRIFUGO_URL", "http://centrifugo:8000")
	t.Setenv("CENTRIFUGO_API_KEY", "apikey")
	t.Setenv("CENTRIFUGO_HMAC_KEY", "hmackey")
	t.Setenv("ORY_KRATOS_URL", "http://kratos:4433")
}

func TestLoad_Defaults(t *testing.T) {
	validEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "canvas.strokes", cfg.StrokeTopic)
	assert.Equal(t, "canvas.hints", cfg.HintTopic)
	assert.Equal(t, "canvas-api", cfg.KafkaConsumerGroup)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, "canvas-api", cfg.ServiceName)
	assert.Equal(t, 1.0, cfg.OTelSampleRate)
	assert.Equal(t, 5*time.Second, cfg.ReadHeaderTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.IdleTimeout)
	assert.Equal(t, 10*time.Second, cfg.ShutdownTimeout)
}

func TestLoad_BrokersFromEnv(t *testing.T) {
	validEnv(t)
	t.Setenv("AUTOMQ_BROKERS", "a:9092,b:9092,c:9092")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, []string{"a:9092", "b:9092", "c:9092"}, cfg.AutoMQBrokers)
}

func TestLoad_AllowedOriginsFromEnv(t *testing.T) {
	validEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, []string{"https://app.example.com", "https://admin.example.com"}, cfg.AllowedOrigins)
}

func TestLoad_MissingBrokers(t *testing.T) {
	validEnv(t)
	t.Setenv("AUTOMQ_BROKERS", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AUTOMQ_BROKERS")
}

func TestLoad_MissingCentrifugoURL(t *testing.T) {
	validEnv(t)
	t.Setenv("CENTRIFUGO_URL", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CENTRIFUGO_URL")
}

func TestLoad_MissingCentrifugoAPIKey(t *testing.T) {
	validEnv(t)
	t.Setenv("CENTRIFUGO_API_KEY", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CENTRIFUGO_API_KEY")
}

func TestLoad_MissingCentrifugoHMACKey(t *testing.T) {
	validEnv(t)
	t.Setenv("CENTRIFUGO_HMAC_KEY", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CENTRIFUGO_HMAC_KEY")
}

func TestLoad_MissingOryKratosURL(t *testing.T) {
	validEnv(t)
	t.Setenv("ORY_KRATOS_URL", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ORY_KRATOS_URL")
}

// TestLoad_AllowedOriginsDefault verifies that the built-in default is used
// when ALLOWED_ORIGINS is not provided. Because Viper falls back to SetDefault
// when an env var is empty, the validate() check for ALLOWED_ORIGINS is only
// reachable if the key is explicitly set to a non-empty but invalid value,
// which is not a realistic scenario — so we just verify the default is intact.
func TestLoad_AllowedOriginsDefault(t *testing.T) {
	validEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.AllowedOrigins)
}
