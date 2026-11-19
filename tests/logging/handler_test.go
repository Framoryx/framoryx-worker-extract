package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"testing"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/logging"
	"github.com/Framoryx/framoryx-worker-extract/internal/platform/tracecontext"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestAutomaticFieldsFollowActiveSpanAndProtectReservedKeys(t *testing.T) {
	var output bytes.Buffer
	logger := logging.NewJSON(&output, "worker", nil)
	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.NeverSample()))
	defer provider.Shutdown(context.Background())
	ctx, root := provider.Tracer("test").Start(tracecontext.NewAttempt(context.Background()), "root")
	defer root.End()
	ctx, child := provider.Tracer("test").Start(ctx, "child")
	defer child.End()
	logger.With("traceId", "falso", "operation", "extraction").WithGroup("details").With("requestId", "falso", "attempt", 1).InfoContext(ctx, "Extração iniciada", "spanId", "falso", "frames", 5)
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	ids := tracecontext.FromContext(ctx)
	if entry["traceId"] != child.SpanContext().TraceID().String() || entry["spanId"] != child.SpanContext().SpanID().String() || entry["correlationId"] != ids.CorrelationID || entry["requestId"] != ids.RequestID || entry["service"] != "worker" {
		t.Fatalf("identificadores incorretos: %v", entry)
	}
	if trace.SpanContextFromContext(ctx).IsSampled() {
		t.Fatal("teste deveria usar trace não amostrado")
	}
	details := entry["details"].(map[string]any)
	if details["spanId"] != nil || details["frames"] != float64(5) || details["attempt"] != float64(1) || entry["operation"] != "extraction" {
		t.Fatalf("atributos comuns incorretos: %v", entry)
	}
}

func TestStartupAndGroups(t *testing.T) {
	var output bytes.Buffer
	logger := logging.NewJSON(&output, "worker", &slog.HandlerOptions{Level: slog.LevelInfo})
	logger.Debug("Não deve aparecer")
	logger.WithGroup("").WithGroup("traceId").With(slog.Group("requestId", "fake", 1), slog.Group("extra", "service", "falso", "valid", true)).Info("Inicialização")
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"traceId", "spanId", "correlationId", "requestId"} {
		if _, ok := entry[key]; ok {
			t.Fatalf("ID inventado na inicialização: %v", entry)
		}
	}
	if entry["extra"].(map[string]any)["valid"] != true {
		t.Fatal("grupo legítimo descartado")
	}
}

func TestConcurrentContextsRemainIsolatedAfterCancellation(t *testing.T) {
	var output bytes.Buffer
	logger := logging.NewJSON(&output, "worker", nil)
	var workers sync.WaitGroup
	for range 20 {
		ctx, cancel := context.WithCancel(tracecontext.NewAttempt(context.Background()))
		cancel()
		workers.Go(func() {
			logger.InfoContext(ctx, "Operação cancelada")
		})
	}
	workers.Wait()
	decoder := json.NewDecoder(&output)
	seen := map[string]bool{}
	for range 20 {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		id := entry["requestId"].(string)
		if seen[id] || !tracecontext.ValidID(id) {
			t.Fatalf("contexto compartilhado entre operações: %s", id)
		}
		seen[id] = true
	}
}
