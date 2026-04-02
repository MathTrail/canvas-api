package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Config holds all configuration for canvas-api.
// Secrets (CentrifugoAPIKey, CentrifugoHMACKey, KafkaSASLUsername,
// KafkaSASLPassword) are injected from K8s Secrets created by VSO.
type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`

	// AutoMQ / Kafka
	AutoMQBrokers      []string `mapstructure:"AUTOMQ_BROKERS"`
	StrokeTopic        string   `mapstructure:"STROKE_TOPIC"`
	HintTopic          string   `mapstructure:"HINT_TOPIC"`
	KafkaConsumerGroup string   `mapstructure:"KAFKA_CONSUMER_GROUP"`
	KafkaSASLUsername  string   `mapstructure:"KAFKA_SASL_USERNAME"`
	KafkaSASLPassword  string   `mapstructure:"KAFKA_SASL_PASSWORD"`

	// Centrifugo
	CentrifugoURL     string `mapstructure:"CENTRIFUGO_URL"`
	CentrifugoAPIKey  string `mapstructure:"CENTRIFUGO_API_KEY"`
	CentrifugoHMACKey string `mapstructure:"CENTRIFUGO_HMAC_KEY"`

	// Ory Kratos
	OryKratosURL string `mapstructure:"ORY_KRATOS_URL"`

	// CORS: comma-separated list of allowed origins.
	// Must include the ui-web shell origin and localhost:3001 for local dev.
	AllowedOrigins []string `mapstructure:"ALLOWED_ORIGINS"`

	// Logging
	LogLevel  string `mapstructure:"LOG_LEVEL"`
	LogFormat string `mapstructure:"LOG_FORMAT"` // "json" or "console"

	// HTTP server timeouts
	ReadHeaderTimeout time.Duration `mapstructure:"READ_HEADER_TIMEOUT"`
	ReadTimeout       time.Duration `mapstructure:"READ_TIMEOUT"`
	WriteTimeout      time.Duration `mapstructure:"WRITE_TIMEOUT"`
	IdleTimeout       time.Duration `mapstructure:"IDLE_TIMEOUT"`
	ShutdownTimeout   time.Duration `mapstructure:"SHUTDOWN_TIMEOUT"`

	// Observability
	ServiceName       string  `mapstructure:"SERVICE_NAME"`
	OTelEndpoint      string  `mapstructure:"OTEL_ENDPOINT"`
	OTelSampleRate    float64 `mapstructure:"OTEL_SAMPLE_RATE"`
	PyroscopeEndpoint string  `mapstructure:"PYROSCOPE_ENDPOINT"`
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

	cfg := &Config{}
	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))
	if err := v.Unmarshal(cfg, decodeHook); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.AutoMQBrokers) == 0 {
		return errors.New("AUTOMQ_BROKERS must contain at least one broker")
	}
	if len(c.AllowedOrigins) == 0 {
		return errors.New("ALLOWED_ORIGINS must contain at least one origin")
	}
	required := []struct{ name, val string }{
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
