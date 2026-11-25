package logging

import (
	"context"
	"io"
	"log/slog"
	"slices"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/tracecontext"
	"go.opentelemetry.io/otel/trace"
)

func NewJSON(writer io.Writer, service string, options *slog.HandlerOptions) *slog.Logger {
	return slog.New(&Handler{next: slog.NewJSONHandler(writer, options), service: service})
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	output := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	var attrs []slog.Attr
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	output.AddAttrs(cleanAttrs(attrs)...)
	fields := []slog.Attr{slog.String("service", h.service)}
	ids := tracecontext.FromContext(ctx)
	if ids.CorrelationID != "" {
		fields = append(fields, slog.String("correlationId", ids.CorrelationID))
	}
	if ids.RequestID != "" {
		fields = append(fields, slog.String("requestId", ids.RequestID))
	}
	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		fields = append(fields, slog.String("traceId", span.TraceID().String()), slog.String("spanId", span.SpanID().String()))
	}
	next := h.next.WithAttrs(fields)
	for _, step := range h.steps {
		if step.group != "" {
			next = next.WithGroup(step.group)
		} else {
			next = next.WithAttrs(step.attrs)
		}
	}
	return next.Handle(ctx, output)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.steps = append(slices.Clone(h.steps), handlerStep{attrs: cleanAttrs(attrs)})
	return &clone
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" || reservedKey(name) {
		return h
	}
	clone := *h
	clone.steps = append(slices.Clone(h.steps), handlerStep{group: name})
	return &clone
}

func cleanAttrs(attrs []slog.Attr) []slog.Attr {
	var clean []slog.Attr
	for _, attr := range attrs {
		if reservedKey(attr.Key) {
			continue
		}
		attr.Value = attr.Value.Resolve()
		if attr.Value.Kind() == slog.KindGroup {
			attr = slog.GroupAttrs(attr.Key, cleanAttrs(attr.Value.Group())...)
		}
		clean = append(clean, attr)
	}
	return clean
}

func reservedKey(key string) bool {
	switch key {
	case "traceId", "spanId", "correlationId", "requestId", "service":
		return true
	default:
		return false
	}
}
