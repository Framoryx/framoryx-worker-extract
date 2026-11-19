package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

func (t *Telemetry) registerMetrics() error {
	meter := t.meterProvider.Meter(ServiceName)
	var err error

	t.jobs, err = meter.Int64Counter(MetricPrefix+"extraction.jobs",
		metric.WithUnit("{job}"),
		metric.WithDescription("Total de extrações finalizadas"),
	)
	if err != nil {
		return fmt.Errorf("criar contador de extrações: %w", err)
	}

	t.failures, err = meter.Int64Counter(MetricPrefix+"extraction.failures",
		metric.WithUnit("{job}"),
		metric.WithDescription("Total de extrações com falha"),
	)
	if err != nil {
		return fmt.Errorf("criar contador de falhas: %w", err)
	}

	t.activeJobs, err = meter.Int64UpDownCounter(MetricPrefix+"extraction.active_jobs",
		metric.WithUnit("{job}"),
		metric.WithDescription("Extrações em andamento"),
	)
	if err != nil {
		return fmt.Errorf("criar contador de extrações em andamento: %w", err)
	}

	t.duration, err = meter.Float64Histogram(MetricPrefix+"extraction.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duração das extrações"),
		metric.WithExplicitBucketBoundaries(1, 5, 15, 30, 60, 120, 300, 600, 1200, 1800, 3600),
	)
	if err != nil {
		return fmt.Errorf("criar histograma de duração: %w", err)
	}

	t.frames, err = meter.Int64Counter(MetricPrefix+"extraction.frames",
		metric.WithUnit("{frame}"),
		metric.WithDescription("Total de frames extraídos com sucesso"),
	)
	if err != nil {
		return fmt.Errorf("criar contador de frames: %w", err)
	}

	t.archiveSize, err = meter.Int64Histogram(MetricPrefix+"archive.size",
		metric.WithUnit("By"),
		metric.WithDescription("Tamanho dos arquivos ZIP gerados"),
		metric.WithExplicitBucketBoundaries(1048576, 10485760, 104857600, 524288000, 1073741824, 5368709120),
	)
	if err != nil {
		return fmt.Errorf("criar histograma de tamanho do ZIP: %w", err)
	}
	return nil
}
