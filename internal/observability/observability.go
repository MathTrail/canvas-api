package observability

import (
	"context"
	"fmt"

	"github.com/grafana/pyroscope-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mathtrail/canvas-api/internal/config"
)

// Observability manages the lifecycle of tracing, metrics, and profiling.
type Observability struct {
	cfg             *config.Config
	logger          *zap.Logger
	tracerShutdown  func(context.Context) error
	metricsShutdown func(context.Context) error
	profiler        interface{ Stop() error }
}

// New creates an Observability manager.
func New(cfg *config.Config, logger *zap.Logger) *Observability {
	return &Observability{cfg: cfg, logger: logger}
}

// Init initialises all configured observability components.
func (o *Observability) Init(ctx context.Context) error {
	if o.cfg.OTelEndpoint != "" {
		shutdown, err := initTracer(ctx, o.cfg)
		if err != nil {
			return fmt.Errorf("init tracer: %w", err)
		}
		o.tracerShutdown = shutdown
	}

	metricsShutdown, err := initMetrics()
	if err != nil {
		return fmt.Errorf("init metrics: %w", err)
	}
	o.metricsShutdown = metricsShutdown

	if o.cfg.PyroscopeEndpoint != "" {
		p, err := initPyroscope(o.cfg)
		if err != nil {
			return fmt.Errorf("init pyroscope: %w", err)
		}
		o.profiler = p
	}

	o.logger.Info("observability stack initialized",
		zap.Bool("tracing", o.cfg.OTelEndpoint != ""),
		zap.Bool("profiling", o.cfg.PyroscopeEndpoint != ""),
	)
	return nil
}

// Shutdown gracefully stops all observability components.
// The caller owns the context and its deadline.
// Order: metrics → profiler → tracer (tracer last so final spans from other
// components are still flushed).
func (o *Observability) Shutdown(ctx context.Context) {
	if o.metricsShutdown != nil {
		if err := o.metricsShutdown(ctx); err != nil {
			o.logger.Warn("metrics shutdown error", zap.Error(err))
		}
	}
	if o.profiler != nil {
		if err := o.profiler.Stop(); err != nil {
			o.logger.Warn("profiler stop error", zap.Error(err))
		}
	}
	if o.tracerShutdown != nil {
		if err := o.tracerShutdown(ctx); err != nil {
			o.logger.Warn("tracer shutdown error", zap.Error(err))
		}
	}

	o.logger.Info("observability stack shut down")
}

// initTracer initialises the global OTLP gRPC trace exporter and TracerProvider.
// It sets the global W3C TraceContext + Baggage propagator so that traceparent
// headers are automatically extracted and Go spans appear as children of upstream
// spans in Tempo.
func initTracer(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	conn, err := grpc.NewClient(
		cfg.OTelEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Use a fresh context so that early cancellation of the application root
	// context does not abort resource detection during startup.
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.K8SPodName(cfg.PodName),
			semconv.K8SNamespaceName(cfg.Namespace),
		),
		resource.WithFromEnv(), // picks up OTEL_RESOURCE_ATTRIBUTES if set
	)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	// ParentBased respects upstream sampling decisions while
	// TraceIDRatioBased controls the rate for locally-originated root spans.
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.OTelSampleRate))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxQueueSize(2048),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C traceparent
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		errTP := tp.Shutdown(ctx)
		errConn := conn.Close()
		if errTP != nil {
			return errTP
		}
		return errConn
	}, nil
}

// initMetrics initialises the global Prometheus MeterProvider.
// The Prometheus exporter registers with prometheus.DefaultRegisterer so
// promhttp.Handler() in the router automatically exposes all OTel metrics.
func initMetrics() (shutdown func(context.Context) error, err error) {
	exporter, err := promexporter.New()
	if err != nil {
		return nil, err
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(exporter),
	)

	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

// initPyroscope starts the Pyroscope continuous profiling agent.
// It collects CPU, allocation and in-use heap profiles and pushes them to the
// Pyroscope server.
func initPyroscope(cfg *config.Config) (*pyroscope.Profiler, error) {
	return pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ServiceName,
		ServerAddress:   cfg.PyroscopeEndpoint,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
		},
	})
}
