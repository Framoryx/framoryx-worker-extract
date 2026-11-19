package telemetry_test

import (
	"context"
	"github.com/Framoryx/framoryx-worker-extract/internal/platform/telemetry"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func preserveGlobals(t *testing.T) {
	t.Helper()
	meter := otel.GetMeterProvider()
	tracer := otel.GetTracerProvider()
	propagator := otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetMeterProvider(meter)
		otel.SetTracerProvider(tracer)
		otel.SetTextMapPropagator(propagator)
	})
}

func TestNewExportsOTLPAndPropagatesContext(t *testing.T) {
	preserveGlobals(t)
	requests := map[string]chan []byte{
		"/v1/metrics": make(chan []byte, 2),
		"/v1/traces":  make(chan []byte, 2),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		channel, ok := requests[r.URL.Path]
		if !ok || r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/x-protobuf" {
			t.Errorf("requisição OTLP inválida: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		channel <- body
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	for _, key := range []string{
		"OTEL_EXPORTER_OTLP_HEADERS", "OTEL_EXPORTER_OTLP_CERTIFICATE",
		"OTEL_EXPORTER_OTLP_CLIENT_CERTIFICATE", "OTEL_EXPORTER_OTLP_CLIENT_KEY",
		"OTEL_RESOURCE_ATTRIBUTES", "OTEL_SERVICE_NAME",
	} {
		t.Setenv(key, "")
	}
	for _, signal := range []string{"METRICS", "TRACES"} {
		t.Setenv("OTEL_EXPORTER_OTLP_"+signal+"_ENDPOINT", server.URL+"/v1/"+strings.ToLower(signal))
		for _, suffix := range []string{"HEADERS", "CERTIFICATE", "CLIENT_CERTIFICATE", "CLIENT_KEY", "COMPRESSION"} {
			t.Setenv("OTEL_EXPORTER_OTLP_"+signal+"_"+suffix, "")
		}
	}
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "3600000")
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	previousMeter := otel.GetMeterProvider()
	previousTracer := otel.GetTracerProvider()
	tel, err := telemetry.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tel.Shutdown(ctx)
	if otel.GetMeterProvider() == previousMeter || otel.GetTracerProvider() == previousTracer {
		t.Fatal("providers globais não configurados")
	}
	member, err := baggage.NewMember("operation", "extraction")
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	extractionCtx, finish := tel.BeginExtraction(baggage.ContextWithBaggage(ctx, bag))
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(extractionCtx, carrier)
	propagated := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
	if trace.SpanContextFromContext(propagated).TraceID() != trace.SpanContextFromContext(extractionCtx).TraceID() || baggage.FromContext(propagated).Member("operation").Value() != "extraction" {
		t.Fatal("trace e baggage não foram propagados")
	}
	finish(telemetry.ExtractionResult{FrameCount: 3, ArchiveSizeBytes: 512})
	if err := tel.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	for path, channel := range requests {
		select {
		case body := <-channel:
			if len(body) == 0 || !strings.Contains(string(body), telemetry.ServiceName) {
				t.Errorf("payload de %s sem identificação do serviço", path)
			}
			if path == "/v1/metrics" && !strings.Contains(string(body), "framoryx.app.extraction.jobs") {
				t.Error("payload de métricas sem prefixo obrigatório")
			}
		case <-ctx.Done():
			t.Fatalf("exportação não recebida em %s: %v", path, ctx.Err())
		}
	}
}

func TestNewRejectsInvalidResource(t *testing.T) {
	preserveGlobals(t)
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "invalid")
	meter := otel.GetMeterProvider()
	tracer := otel.GetTracerProvider()
	tel, err := telemetry.New(context.Background())
	if err == nil || tel != nil || !strings.Contains(err.Error(), "criar recurso de telemetria") {
		t.Fatalf("recurso inválido foi aceito: %v, %v", tel, err)
	}
	if otel.GetMeterProvider() != meter || otel.GetTracerProvider() != tracer {
		t.Error("falha de inicialização alterou providers globais")
	}
}
