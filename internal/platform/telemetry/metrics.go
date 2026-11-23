package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

func (t *Telemetry) registerMetrics() error {
	meter := t.meterProvider.Meter(ServiceName)
	initializers := []func() error{
		metricInitializer(&t.jobs, "criar contador de extrações", func() (metric.Int64Counter, error) {
			return meter.Int64Counter(MetricPrefix+"extraction.jobs",
				metric.WithUnit("{job}"),
				metric.WithDescription("Total de extrações finalizadas"),
			)
		}),
		metricInitializer(&t.failures, "criar contador de falhas", func() (metric.Int64Counter, error) {
			return meter.Int64Counter(MetricPrefix+"extraction.failures",
				metric.WithUnit("{job}"),
				metric.WithDescription("Total de extrações com falha"),
			)
		}),
		metricInitializer(&t.activeJobs, "criar contador de extrações em andamento", func() (metric.Int64UpDownCounter, error) {
			return meter.Int64UpDownCounter(MetricPrefix+"extraction.active_jobs",
				metric.WithUnit("{job}"),
				metric.WithDescription("Extrações em andamento"),
			)
		}),
		metricInitializer(&t.duration, "criar histograma de duração", func() (metric.Float64Histogram, error) {
			return meter.Float64Histogram(MetricPrefix+"extraction.duration",
				metric.WithUnit("s"),
				metric.WithDescription("Duração das extrações"),
				metric.WithExplicitBucketBoundaries(1, 5, 15, 30, 60, 120, 300, 600, 1200, 1800, 3600),
			)
		}),
		metricInitializer(&t.frames, "criar contador de frames", func() (metric.Int64Counter, error) {
			return meter.Int64Counter(MetricPrefix+"extraction.frames",
				metric.WithUnit("{frame}"),
				metric.WithDescription("Total de frames extraídos com sucesso"),
			)
		}),
		metricInitializer(&t.archiveSize, "criar histograma de tamanho do ZIP", func() (metric.Int64Histogram, error) {
			return meter.Int64Histogram(MetricPrefix+"archive.size",
				metric.WithUnit("By"),
				metric.WithDescription("Tamanho dos arquivos ZIP gerados"),
				metric.WithExplicitBucketBoundaries(1048576, 10485760, 104857600, 524288000, 1073741824, 5368709120),
			)
		}),
	}
	for _, initialize := range initializers {
		if err := initialize(); err != nil {
			return err
		}
	}
	return nil
}

func metricInitializer[T any](target *T, operation string, create func() (T, error)) func() error {
	return func() error {
		value, err := create()
		*target = value
		if err != nil {
			return fmt.Errorf("%s: %w", operation, err)
		}
		return nil
	}
}
