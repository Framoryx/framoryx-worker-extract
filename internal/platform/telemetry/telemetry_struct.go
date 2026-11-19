package telemetry

import (
	"sync"

	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	meterProvider  *sdkmetric.MeterProvider
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	jobs           metric.Int64Counter
	failures       metric.Int64Counter
	activeJobs     metric.Int64UpDownCounter
	duration       metric.Float64Histogram
	frames         metric.Int64Counter
	archiveSize    metric.Int64Histogram
	shutdownOnce   sync.Once
	shutdownErr    error
}
