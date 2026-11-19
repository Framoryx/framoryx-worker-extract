package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	ServiceName  = "framoryx-worker-extract"
	MetricPrefix = "framoryx.app."
)

func New(ctx context.Context) (*Telemetry, error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", ServiceName),
			attribute.String("service.namespace", "framoryx"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("criar recurso de telemetria: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("criar exportador de métricas: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return nil, errors.Join(
			fmt.Errorf("criar exportador de traces: %w", err),
			metricExporter.Shutdown(cleanupCtx),
		)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)

	t, err := NewWithProviders(meterProvider, tracerProvider)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return nil, errors.Join(err, meterProvider.Shutdown(cleanupCtx), tracerProvider.Shutdown(cleanupCtx))
	}

	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	return t, nil
}

func NewWithProviders(meterProvider *sdkmetric.MeterProvider, tracerProvider *sdktrace.TracerProvider) (*Telemetry, error) {
	t := &Telemetry{
		meterProvider:  meterProvider,
		tracerProvider: tracerProvider,
		tracer:         tracerProvider.Tracer(ServiceName),
	}
	if err := t.registerMetrics(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	t.shutdownOnce.Do(func() {
		t.shutdownErr = errors.Join(
			t.meterProvider.Shutdown(ctx),
			t.tracerProvider.Shutdown(ctx),
		)
	})
	return t.shutdownErr
}
