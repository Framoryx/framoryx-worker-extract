package tracecontext

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const CorrelationHeader = "x-correlation-id"

func ExtractMessage(ctx context.Context, carrier propagation.TextMapCarrier, trusted bool) context.Context {
	ctx = context.WithValue(ctx, identifiersKey, Identifiers{})
	ctx = trace.ContextWithSpanContext(ctx, trace.SpanContext{})
	if carrier == nil {
		return ctx
	}
	ctx = (propagation.TraceContext{}).Extract(ctx, carrier)
	if trusted {
		value := carrier.Get(CorrelationHeader)
		if multi, ok := carrier.(propagation.ValuesGetter); ok && len(multi.Values(CorrelationHeader)) != 1 {
			value = ""
		}
		if ValidID(value) {
			ctx = context.WithValue(ctx, identifiersKey, Identifiers{CorrelationID: value})
		}
	}
	return ctx
}

func InjectMessage(ctx context.Context, carrier propagation.TextMapCarrier) {
	if carrier == nil {
		return
	}
	(propagation.TraceContext{}).Inject(ctx, carrier)
	if ids := FromContext(ctx); ValidID(ids.CorrelationID) {
		carrier.Set(CorrelationHeader, ids.CorrelationID)
	}
}
