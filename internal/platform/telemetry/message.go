package telemetry

import (
	"context"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/tracecontext"
	"go.opentelemetry.io/otel/propagation"
)

func (t *Telemetry) BeginMessageExtraction(ctx context.Context, carrier propagation.TextMapCarrier, trusted bool) (context.Context, func(ExtractionResult)) {
	return t.BeginExtraction(tracecontext.ExtractMessage(ctx, carrier, trusted))
}
