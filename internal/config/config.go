package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for canvas-api.
// Secrets (CentrifugoAPIKey, CentrifugoHMACKey, KafkaSASLUsername,
// KafkaSASLPassword) are injected from K8s Secrets created by VSO.
type Config struct {
	ServerPort string

	// AutoMQ / Kafka
	AutoMQBrokers      []string
	StrokeTopic        string
	HintTopic          string
	KafkaConsumerGroup string
	KafkaSASLUsername  string
	KafkaSASLPassword  string

	// Centrifugo
	CentrifugoURL     string
	CentrifugoAPIKey  string
	CentrifugoHMACKey string

	// Ory Kratos
	OryKratosURL string

	// CORS: comma-separated list of allowed origins.
	// Must include the ui-web shell origin and localhost:3001 for local dev.
	AllowedOrigins []string

	// Logging
	LogLevel  string
	LogFormat string // "json" or "console"

	// HTTP server timeouts
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration

	// Observability
	ServiceName       string
	OTelEndpoint      string
	OTelSampleRate    float64
	PyroscopeEndpoint string
}

func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("SERVER_PORT", "8080")
	v.SetDefault("STROKE_TOPIC", "canvas.strokes")
	v.SetDefault("HINT_TOPIC", "canvas.hints")
	v.SetDefault("KAFKA_CONSUMER_GROUP", "canvas-api")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
	v.SetDefault("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001")
	v.SetDefault("READ_HEADER_TIMEOUT", "5s")
	v.SetDefault("READ_TIMEOUT", "5s")
	v.SetDefault("WRITE_TIMEOUT", "10s")
	v.SetDefault("IDLE_TIMEOUT", "120s")
	v.SetDefault("SHUTDOWN_TIMEOUT", "10s")
	v.SetDefault("SERVICE_NAME", "canvas-api")
	v.SetDefault("OTEL_SAMPLE_RATE", 1.0)

	cfg := &Config{
		ServerPort:         v.GetString("SERVER_PORT"),
		StrokeTopic:        v.GetString("STROKE_TOPIC"),
		HintTopic:          v.GetString("HINT_TOPIC"),
		KafkaConsumerGroup: v.GetString("KAFKA_CONSUMER_GROUP"),
		KafkaSASLUsername:  v.GetString("KAFKA_SASL_USERNAME"),
		KafkaSASLPassword:  v.GetString("KAFKA_SASL_PASSWORD"),
		CentrifugoURL:      v.GetString("CENTRIFUGO_URL"),
		CentrifugoAPIKey:   v.GetString("CENTRIFUGO_API_KEY"),
		CentrifugoHMACKey:  v.GetString("CENTRIFUGO_HMAC_KEY"),
		OryKratosURL:       v.GetString("ORY_KRATOS_URL"),
		LogLevel:           v.GetString("LOG_LEVEL"),
		LogFormat:          v.GetString("LOG_FORMAT"),
		ReadHeaderTimeout:  v.GetDuration("READ_HEADER_TIMEOUT"),
		ReadTimeout:        v.GetDuration("READ_TIMEOUT"),
		WriteTimeout:       v.GetDuration("WRITE_TIMEOUT"),
		IdleTimeout:        v.GetDuration("IDLE_TIMEOUT"),
		ShutdownTimeout:    v.GetDuration("SHUTDOWN_TIMEOUT"),
		ServiceName:        v.GetString("SERVICE_NAME"),
		OTelEndpoint:       v.GetString("OTEL_ENDPOINT"),
		OTelSampleRate:     v.GetFloat64("OTEL_SAMPLE_RATE"),
		PyroscopeEndpoint:  v.GetString("PYROSCOPE_ENDPOINT"),
	}

	// CSV-split slice fields
	if raw := v.GetString("AUTOMQ_BROKERS"); raw != "" {
		cfg.AutoMQBrokers = strings.Split(raw, ",")
	}
	if raw := v.GetString("ALLOWED_ORIGINS"); raw != "" {
		cfg.AllowedOrigins = strings.Split(raw, ",")
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	required := []struct{ name, val string }{
		{"AUTOMQ_BROKERS", strings.Join(c.AutoMQBrokers, ",")},
		{"CENTRIFUGO_URL", c.CentrifugoURL},
		{"CENTRIFUGO_API_KEY", c.CentrifugoAPIKey},
		{"CENTRIFUGO_HMAC_KEY", c.CentrifugoHMACKey},
		{"ORY_KRATOS_URL", c.OryKratosURL},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("required env var %s is not set", r.name)
		}
	}
	return nil
}
