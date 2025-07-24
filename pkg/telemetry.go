package pkg

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
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
		),
	)

	if err != nil {
		panic(err)
	}

	return &Telemetry{
		resource: res,
	}
}

func (t *Telemetry) initTraceExporter(ctx context.Context) error {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())

	if err != nil {
		return err
	}

	t.traceExporter = exporter

	return nil
}

func (t *Telemetry) initMeterExporter(ctx context.Context) error {
	exporter, err := otlpmetricgrpc.New(ctx)
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

func (t *Telemetry) Init(ctx context.Context) error {
	if err := t.initTraceExporter(ctx); err != nil {
		return err
	}

	if err := t.initMeterExporter(ctx); err != nil {
		return err
	}

	t.initTraceProvider()
	t.initMeterProvider()

	return nil
}

func (t *Telemetry) Close() {
	t.traceProvider.Shutdown(context.Background())
	t.meterProvider.Shutdown(context.Background())
}
