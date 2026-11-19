package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/logging"
	"github.com/Framoryx/framoryx-worker-extract/internal/platform/telemetry"
	"github.com/Framoryx/framoryx-worker-extract/internal/platform/tracecontext"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestMessageRedeliveryPreservesFlowAndCreatesNewAttempt(t *testing.T) {
	tel, reader, exporter := testTelemetry(t)
	producerCtx, producer := tel.Tracer().Start(tracecontext.NewAttempt(context.Background()), "publish")
	defer producer.End()
	carrier := propagation.MapCarrier{}
	tracecontext.InjectMessage(producerCtx, carrier)
	var output bytes.Buffer
	logger := logging.NewJSON(&output, telemetry.ServiceName, nil)
	requestIDs := map[string]bool{}
	spanIDs := map[trace.SpanID]bool{}
	for range 2 {
		ctx, finish := tel.BeginMessageExtraction(context.Background(), carrier, true)
		ids := tracecontext.FromContext(ctx)
		span := trace.SpanContextFromContext(ctx)
		if ids.CorrelationID != tracecontext.FromContext(producerCtx).CorrelationID || requestIDs[ids.RequestID] || spanIDs[span.SpanID()] || span.TraceID() != producer.SpanContext().TraceID() {
			t.Fatal("reentrega perdeu correlação ou reutilizou tentativa")
		}
		requestIDs[ids.RequestID] = true
		spanIDs[span.SpanID()] = true
		logger.InfoContext(ctx, "Mensagem recebida")
		finish(telemetry.ExtractionResult{Err: errors.New("falha temporária")})
	}
	decoder := json.NewDecoder(&output)
	for range 2 {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		if !requestIDs[entry["requestId"].(string)] || entry["traceId"] != producer.SpanContext().TraceID().String() {
			t.Fatalf("log sem contexto da tentativa: %v", entry)
		}
	}
	metrics := collectMetrics(t, reader)
	assertSum(t, metrics, "extraction.failures", 2)
	if len(exporter.GetSpans()) != 2 {
		t.Fatal("spans de processamento não encerrados")
	}
}
