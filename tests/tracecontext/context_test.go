package tracecontext_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/tracecontext"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestAttemptsAndPropagation(t *testing.T) {
	ctx := tracecontext.NewAttempt(context.Background())
	first := tracecontext.FromContext(ctx)
	if !tracecontext.ValidID(first.CorrelationID) || !tracecontext.ValidID(first.RequestID) || first.CorrelationID == first.RequestID {
		t.Fatalf("identificadores inválidos: %+v", first)
	}
	carrier := propagation.MapCarrier{}
	tracecontext.InjectMessage(ctx, carrier)
	carrier["x-request-id"] = first.RequestID
	secondCtx := tracecontext.NewAttempt(tracecontext.ExtractMessage(ctx, carrier, true))
	second := tracecontext.FromContext(secondCtx)
	if first.CorrelationID != second.CorrelationID || first.RequestID == second.RequestID {
		t.Fatalf("tentativas sem isolamento: %+v %+v", first, second)
	}
	if tracecontext.FromContext(ctx) != first {
		t.Fatal("contexto pai foi modificado")
	}
	untrusted := tracecontext.FromContext(tracecontext.NewAttempt(tracecontext.ExtractMessage(ctx, carrier, false)))
	if untrusted.CorrelationID == first.CorrelationID {
		t.Fatal("origem não confiável preservada")
	}
	tracecontext.InjectMessage(ctx, nil)
	if tracecontext.FromContext(tracecontext.ExtractMessage(ctx, nil, true)).CorrelationID != "" {
		t.Fatal("contexto anterior vazou sem mensagem")
	}
}

func TestInvalidAndDuplicateIdentifiers(t *testing.T) {
	for _, id := range []string{"", "inválido", strings.Repeat("a", 100), "dc9d13c3-81d0-4e24-8d80-339a995a1fa7\r\n", "DC9D13C3-81D0-4E24-8D80-339A995A1FA7", "00000000-0000-0000-0000-000000000000", "dc9d13c3-81d0-1e24-8d80-339a995a1fa7"} {
		if tracecontext.ValidID(id) {
			t.Fatalf("ID inválido aceito: %q", id)
		}
	}
	header := http.Header{}
	header.Add(tracecontext.CorrelationHeader, "dc9d13c3-81d0-4e24-8d80-339a995a1fa7")
	header.Add(tracecontext.CorrelationHeader, "dc9d13c3-81d0-4e24-8d80-339a995a1fa7")
	ctx := tracecontext.ExtractMessage(context.Background(), propagation.HeaderCarrier(header), true)
	if tracecontext.FromContext(ctx).CorrelationID != "" {
		t.Fatal("IDs duplicados aceitos")
	}
	for _, carrier := range []propagation.MapCarrier{{tracecontext.CorrelationHeader: "inválido"}, {"traceparent": "inválido"}} {
		ctx := tracecontext.NewAttempt(tracecontext.ExtractMessage(context.Background(), carrier, true))
		if !tracecontext.ValidID(tracecontext.FromContext(ctx).CorrelationID) || trace.SpanContextFromContext(ctx).IsValid() {
			t.Fatal("contexto inválido não tratado")
		}
	}
}
