package telemetry_test

import (
	"context"
	"errors"
	"github.com/Framoryx/framoryx-worker-extract/internal/platform/telemetry"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func testTelemetry(t *testing.T) (*telemetry.Telemetry, *sdkmetric.ManualReader, *tracetest.InMemoryExporter) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	exporter := tracetest.NewInMemoryExporter()
	tel, err := telemetry.NewWithProviders(
		sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)),
		sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample())),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tel.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return tel, reader, exporter
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) map[string]metricdata.Metrics {
	t.Helper()
	var data metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &data); err != nil {
		t.Fatal(err)
	}
	metrics := make(map[string]metricdata.Metrics)
	for _, scope := range data.ScopeMetrics {
		for _, m := range scope.Metrics {
			if !strings.HasPrefix(m.Name, "framoryx.app.") {
				t.Errorf("métrica sem prefixo obrigatório: %s", m.Name)
			}
			metrics[m.Name] = m
		}
	}
	return metrics
}

func assertSum(t *testing.T, metrics map[string]metricdata.Metrics, name string, want int64) attribute.Set {
	t.Helper()
	sum, ok := metrics[telemetry.MetricPrefix+name].Data.(metricdata.Sum[int64])
	if !ok || len(sum.DataPoints) != 1 {
		t.Fatalf("contador %s inválido: %+v", name, metrics[telemetry.MetricPrefix+name])
	}
	if sum.DataPoints[0].Value != want {
		t.Errorf("contador %s: obtido %d, esperado %d", name, sum.DataPoints[0].Value, want)
	}
	return sum.DataPoints[0].Attributes
}

func TestExtractionSuccess(t *testing.T) {
	tel, reader, exporter := testTelemetry(t)
	parentCtx, parent := tel.Tracer().Start(context.Background(), "parent")
	ctx, finish := tel.BeginExtraction(parentCtx)
	assertSum(t, collectMetrics(t, reader), "extraction.active_jobs", 1)
	if !trace.SpanContextFromContext(ctx).IsValid() {
		t.Fatal("contexto da extração sem span válido")
	}
	finish(telemetry.ExtractionResult{FrameCount: 12, ArchiveSizeBytes: 2048})
	metrics := collectMetrics(t, reader)
	attrs := assertSum(t, metrics, "extraction.jobs", 1)
	status, _ := attrs.Value("status")
	if status.AsString() != "success" || attrs.Len() != 1 {
		t.Errorf("atributos de sucesso inválidos: %v", attrs)
	}
	assertSum(t, metrics, "extraction.frames", 12)
	assertSum(t, metrics, "extraction.active_jobs", 0)
	if _, ok := metrics[telemetry.MetricPrefix+"extraction.failures"]; ok {
		t.Error("sucesso contabilizado como falha")
	}
	duration := metrics[telemetry.MetricPrefix+"extraction.duration"].Data.(metricdata.Histogram[float64])
	if len(duration.DataPoints) != 1 || duration.DataPoints[0].Count != 1 || duration.DataPoints[0].Sum < 0 || metrics[telemetry.MetricPrefix+"extraction.duration"].Unit != "s" {
		t.Errorf("duração inválida: %+v", duration)
	}
	archive := metrics[telemetry.MetricPrefix+"archive.size"].Data.(metricdata.Histogram[int64])
	if len(archive.DataPoints) != 1 || archive.DataPoints[0].Sum != 2048 || metrics[telemetry.MetricPrefix+"archive.size"].Unit != "By" {
		t.Errorf("tamanho do ZIP inválido: %+v", archive)
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Name != "extraction.execute" || spans[0].SpanKind != trace.SpanKindConsumer || spans[0].Parent.SpanID() != parent.SpanContext().SpanID() || spans[0].Status.Code == codes.Error {
		t.Fatalf("span de sucesso inválido: %+v", spans)
	}
	parent.End()
}

func TestExtractionFailureFinishesOnceConcurrently(t *testing.T) {
	tel, reader, exporter := testTelemetry(t)
	ctx, cancel := context.WithCancel(context.Background())
	_, finish := tel.BeginExtraction(ctx)
	cancel()
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			finish(telemetry.ExtractionResult{FrameCount: 10, ArchiveSizeBytes: 1024, Err: errors.New("falha no processamento")})
		})
	}
	workers.Wait()
	metrics := collectMetrics(t, reader)
	attrs := assertSum(t, metrics, "extraction.jobs", 1)
	status, _ := attrs.Value("status")
	if status.AsString() != "failure" {
		t.Errorf("status da falha inválido: %v", status)
	}
	assertSum(t, metrics, "extraction.failures", 1)
	assertSum(t, metrics, "extraction.active_jobs", 0)
	for _, name := range []string{"extraction.frames", "archive.size"} {
		if _, ok := metrics[telemetry.MetricPrefix+name]; ok {
			t.Errorf("resultado incompleto contabilizado em %s", name)
		}
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Status.Code != codes.Error || len(spans[0].Events) != 1 {
		t.Fatalf("span de falha inválido: %+v", spans)
	}
}

func TestExtractionIgnoresNonPositiveResults(t *testing.T) {
	tel, reader, _ := testTelemetry(t)
	for _, result := range []telemetry.ExtractionResult{{}, {FrameCount: -1, ArchiveSizeBytes: -1}} {
		_, finish := tel.BeginExtraction(context.Background())
		finish(result)
	}
	metrics := collectMetrics(t, reader)
	assertSum(t, metrics, "extraction.jobs", 2)
	assertSum(t, metrics, "extraction.active_jobs", 0)
	for _, name := range []string{"extraction.frames", "archive.size"} {
		if _, ok := metrics[telemetry.MetricPrefix+name]; ok {
			t.Errorf("valor não positivo contabilizado em %s", name)
		}
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	tel, reader, _ := testTelemetry(t)
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Fatalf("segunda finalização falhou: %v", err)
	}
	var data metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &data); !errors.Is(err, sdkmetric.ErrReaderShutdown) {
		t.Fatalf("leitor não foi encerrado: %v", err)
	}
}
