module github.com/mathtrail/canvas-api

go 1.26.0

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/mathtrail/contracts v0.0.0
	github.com/rs/cors v1.11.1
	github.com/spf13/viper v1.19.0
	github.com/twmb/franz-go v1.18.1
	github.com/twmb/franz-go/pkg/sasl/scram v1.18.1
	go.uber.org/zap v1.27.0
	golang.org/x/sync v0.12.0
	google.golang.org/protobuf v1.36.5
)

replace github.com/mathtrail/contracts => ../contracts
