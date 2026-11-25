package extraction_test

import (
	"context"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// Read exposes the simulated stream until its owner closes it.
func (r *ownedReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, io.ErrClosedPipe
	}
	return r.reader.Read(p)
}

// Close records and closes the simulated owned stream exactly once.
func (r *ownedReader) Close() error {
	if r.closed {
		return io.ErrClosedPipe
	}
	r.closed = true
	return r.closeFn()
}

// Acquire simulates an authorization response and records request identity.
func (s *executionScenario) Acquire(ctx context.Context, request application.ExecutionRequest) (application.Assignment, error) {
	s.step(ctx, "adquirir")
	if request != s.request {
		s.t.Errorf("identidade de aquisição alterada: %+v", request)
	}
	return s.assignment, s.acquireErr
}

// Complete simulates durable acceptance or rejection of the current attempt.
func (s *executionScenario) Complete(ctx context.Context, result domain.ExtractionResult) error {
	s.step(ctx, "confirmar")
	if result.JobID != s.request.JobID || result.VideoID != s.assignment.Job.Source.VideoID || result.AttemptID != s.assignment.Job.AttemptID {
		s.t.Errorf("identidade do resultado alterada: %+v", result)
	}
	if result.Archive != s.receipt.Object || result.ArchiveSizeBytes != s.receipt.SizeBytes || result.FrameCount != s.frameCount || result.ProcessingDuration != s.duration {
		s.t.Errorf("medições ou referência do resultado alteradas: %+v", result)
	}
	return s.completeErr
}

// Open simulates an owned source stream and optional partial failure.
func (s *executionScenario) Open(ctx context.Context, reference domain.ObjectReference) (io.ReadCloser, error) {
	s.step(ctx, "abrir")
	if reference != s.assignment.Job.Source.Object {
		s.t.Errorf("origem fixada alterada: %+v", reference)
	}
	if s.mutateAssigned {
		s.assignment.Job.Options.Segments[0].End = 99 * time.Second
	}
	if s.sourceNil {
		return nil, s.openErr
	}
	return &ownedReader{reader: strings.NewReader("vídeo"), closeFn: func() error {
		s.events = append(s.events, "fechar origem")
		return s.sourceCloseErr
	}}, s.openErr
}

// Process simulates archive production without decoding any real media.
func (s *executionScenario) Process(ctx context.Context, source io.Reader, options domain.ExtractionOptions) (application.ProcessedArchive, error) {
	s.step(ctx, "processar")
	if !reflect.DeepEqual(options, s.expectedOptions) {
		s.t.Errorf("parâmetros autorizados alterados: %+v", options)
	}
	data, err := io.ReadAll(source)
	if err != nil || string(data) != "vídeo" {
		s.t.Errorf("origem indisponível ao processador: %q, %v", data, err)
	}
	archive := application.ProcessedArchive{FrameCount: s.frameCount, ProcessingDuration: s.duration}
	if !s.archiveNil {
		archive.Body = &ownedReader{reader: strings.NewReader("zip"), closeFn: func() error {
			s.events = append(s.events, "fechar arquivo")
			return s.archiveCloseErr
		}}
	}
	return archive, s.processErr
}

// Store consumes the generated stream without taking ownership of it.
func (s *executionScenario) Store(ctx context.Context, destination domain.ObjectReference, body io.Reader) (application.UploadReceipt, error) {
	s.step(ctx, "armazenar")
	if destination != s.assignment.Destination {
		s.t.Errorf("destino autorizado alterado: %+v", destination)
	}
	data, err := io.ReadAll(body)
	if err != nil || string(data) != "zip" {
		s.t.Errorf("arquivo indisponível ao armazenamento: %q, %v", data, err)
	}
	return s.receipt, s.storeErr
}

func (s *executionScenario) step(ctx context.Context, name string) {
	s.t.Helper()
	s.events = append(s.events, name)
	if ctx != s.ctx || ctx.Err() != nil {
		s.t.Errorf("contexto incorreto na operação %s: %v", name, ctx.Err())
	}
	if s.cancelAt == name {
		s.cancel()
	}
}

func newScenario(t *testing.T) *executionScenario {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	job := domain.ExtractionJob{
		ID: "job-1", AttemptID: "attempt-1",
		Source: domain.VideoSource{VideoID: "video-1", Object: domain.ObjectReference{Bucket: "videos", Key: "original.mp4", VersionID: "source-version"}},
		Options: domain.ExtractionOptions{
			Mode: domain.ExtractionModeSampling, FrameInterval: time.Second, ImageFormat: domain.ImageFormatPNG,
			Segments: []domain.VideoSegment{{Start: 0, End: 10 * time.Second}},
		},
	}
	destination := domain.ObjectReference{Bucket: "archives", Key: "job-1/attempt-1.zip"}
	options := job.Options
	options.Segments = slices.Clone(options.Segments)
	return &executionScenario{
		t: t, ctx: ctx, cancel: cancel,
		request:    application.ExecutionRequest{JobID: job.ID, AcquisitionKey: "acquire-1"},
		assignment: application.Assignment{Job: &job, Destination: destination},
		receipt: application.UploadReceipt{
			Object: domain.ObjectReference{Bucket: destination.Bucket, Key: destination.Key, VersionID: "archive-version"}, SizeBytes: 3,
		},
		expectedOptions: options, frameCount: 2, duration: time.Second,
	}
}

func completedResult() domain.ExtractionResult {
	return domain.ExtractionResult{
		JobID: "job-1", VideoID: "video-1", AttemptID: "attempt-1",
		Archive:    domain.ObjectReference{Bucket: "archives", Key: "job-1/attempt-1.zip", VersionID: "archive-version"},
		FrameCount: 2, ArchiveSizeBytes: 3, ProcessingDuration: time.Second,
	}
}
