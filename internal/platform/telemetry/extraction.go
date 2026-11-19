package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/Framoryx/framoryx-worker-extract/internal/platform/tracecontext"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func (t *Telemetry) BeginExtraction(ctx context.Context) (context.Context, func(ExtractionResult)) {
	ctx = tracecontext.NewAttempt(ctx)
	ctx, span := t.tracer.Start(ctx, "extraction.execute", trace.WithSpanKind(trace.SpanKindConsumer))
	startedAt := time.Now()
	t.activeJobs.Add(ctx, 1)
	var once sync.Once

	return ctx, func(result ExtractionResult) {
		once.Do(func() {
			status := "success"
			if result.Err != nil {
				status = "failure"
				t.failures.Add(ctx, 1)
				span.RecordError(result.Err)
				span.SetStatus(codes.Error, "falha na extração")
			} else {
				if result.FrameCount > 0 {
					t.frames.Add(ctx, result.FrameCount)
				}
				if result.ArchiveSizeBytes > 0 {
					t.archiveSize.Record(ctx, result.ArchiveSizeBytes)
				}
			}

			attrs := metric.WithAttributes(attribute.String("status", status))
			t.jobs.Add(ctx, 1, attrs)
			t.duration.Record(ctx, time.Since(startedAt).Seconds(), attrs)
			span.SetAttributes(
				attribute.String("extraction.status", status),
				attribute.Int64("extraction.frame_count", result.FrameCount),
				attribute.Int64("archive.size_bytes", result.ArchiveSizeBytes),
			)
			t.activeJobs.Add(ctx, -1)
			span.End()
		})
	}
}
