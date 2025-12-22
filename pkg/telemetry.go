package pkg

import (
	"context"
	"log"
	"net"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Telemetry struct {
	resource *resource.Resource

	traceExporter *otlptrace.Exporter
	traceProvider *sdktrace.TracerProvider

	meterExporter *otlpmetricgrpc.Exporter
	meterProvider *sdkmetric.MeterProvider
}

func NewTelemetry(name string) *Telemetry {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceVersion("2.0.0"),
		),
	)

	if err != nil {
		panic(err)
	}

	return &Telemetry{
		resource: res,
	}
}

func (t *Telemetry) isCollectorReachable(endpoint string) bool {
	conn, err := net.DialTimeout("tcp", endpoint, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (t *Telemetry) initTraceExporter(ctx context.Context, conn *grpc.ClientConn) error {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return err
	}

	t.traceExporter = exporter

	return nil
}

func (t *Telemetry) initMeterExporter(ctx context.Context, conn *grpc.ClientConn) error {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return err
	}

	t.meterExporter = exporter

	return nil
}

func (t *Telemetry) initTraceProvider() {
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(t.traceExporter),
		sdktrace.WithResource(t.resource),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.05)),
	)

	t.traceProvider = provider
	otel.SetTracerProvider(provider)
}

func (t *Telemetry) initMeterProvider() {
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(t.resource),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(t.meterExporter, sdkmetric.WithInterval(time.Second)),
		),
	)

	t.meterProvider = provider
	otel.SetMeterProvider(provider)
}

func (t *Telemetry) Init(ctx context.Context, cfg *TelemetryConfig) error {
	endpoint := cfg.CollectorEndpoint

	// Check if collector is healthy using gRPC health check
	if !t.isCollectorReachable(endpoint) {
		log.Print("WARNING: OpenTelemetry collector is not healthy or not reachable (gRPC) at ", endpoint)
		return nil
	}

	// Create gRPC connection
	grpcTransport := grpc.WithTransportCredentials(insecure.NewCredentials())
	grcpConn, err := grpc.NewClient(endpoint, grpcTransport)
	if err != nil {
		return err
	}

	if err := t.initTraceExporter(ctx, grcpConn); err != nil {
		return err
	}

	if err := t.initMeterExporter(ctx, grcpConn); err != nil {
		return err
	}

	t.initTraceProvider()
	t.initMeterProvider()

	// Runtime
	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		panic(err)
	}

	return nil
}

func (t *Telemetry) Close() {
	if t.traceExporter != nil {
		t.traceProvider.Shutdown(context.Background())
	}

	if t.meterExporter != nil {
		t.meterProvider.Shutdown(context.Background())
	}
}
