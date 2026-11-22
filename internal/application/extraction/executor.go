package extraction

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// NewExecutor requires all four execution ports.
func NewExecutor(coordinator Coordinator, source SourceStorage, processor ArchiveProcessor, archives ArchiveStorage) (*Executor, error) {
	if coordinator == nil || source == nil || processor == nil || archives == nil {
		return nil, ErrInvalidDependencies
	}
	return &Executor{coordinator: coordinator, source: source, processor: processor, archives: archives}, nil
}

// Execute acquires, produces, uploads and confirms one result without retrying.
func (e *Executor) Execute(ctx context.Context, request ExecutionRequest) (outcome ExecutionOutcome, err error) {
	if e == nil || e.coordinator == nil || e.source == nil || e.processor == nil || e.archives == nil {
		return outcome, ErrInvalidDependencies
	}
	if ctx == nil || strings.TrimSpace(request.JobID) == "" || strings.TrimSpace(request.AcquisitionKey) == "" {
		return outcome, ErrInvalidRequest
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	assignment, err := e.coordinator.Acquire(ctx, request)
	if err != nil {
		return outcome, fmt.Errorf("adquirir tentativa: %w", err)
	}
	if assignment.Completed != nil {
		result := *assignment.Completed
		if assignment.Job != nil || result.JobID != request.JobID {
			return outcome, ErrInvalidAssignment
		}
		if err := result.Validate(); err != nil {
			return outcome, fmt.Errorf("%w: %w", ErrInvalidAssignment, err)
		}
		return ExecutionOutcome{Result: &result, Confirmed: true}, nil
	}
	if assignment.Job == nil || assignment.Job.ID != request.JobID {
		return outcome, ErrInvalidAssignment
	}
	job := *assignment.Job
	job.Options.Segments = slices.Clone(job.Options.Segments)
	if err := job.Validate(); err != nil {
		return outcome, fmt.Errorf("%w: %w", ErrInvalidAssignment, err)
	}
	if err := assignment.Destination.Validate(); err != nil {
		return outcome, fmt.Errorf("destino autorizado: %w: %w", ErrInvalidAssignment, err)
	}
	if assignment.Destination.Bucket == job.Source.Object.Bucket && assignment.Destination.Key == job.Source.Object.Key {
		return outcome, fmt.Errorf("destino do arquivo coincide com o vídeo original: %w", ErrInvalidAssignment)
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	source, err := e.source.Open(ctx, job.Source.Object)
	if source != nil {
		defer closeResource(source, "fechar vídeo de origem", &err)
	}
	if err != nil {
		return outcome, fmt.Errorf("abrir vídeo de origem: %w", err)
	}
	if source == nil {
		return outcome, ErrInvalidPortResult
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	archive, err := e.processor.Process(ctx, source, job.Options)
	if archive.Body != nil {
		defer closeResource(archive.Body, "fechar arquivo gerado", &err)
	}
	if err != nil {
		return outcome, fmt.Errorf("processar vídeo: %w", err)
	}
	if archive.Body == nil || archive.FrameCount <= 0 || archive.ProcessingDuration < 0 {
		return outcome, ErrInvalidPortResult
	}
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	receipt, err := e.archives.Store(ctx, assignment.Destination, archive.Body)
	if err != nil {
		return outcome, fmt.Errorf("armazenar arquivo gerado: %w", err)
	}
	result := domain.ExtractionResult{
		JobID:              job.ID,
		VideoID:            job.Source.VideoID,
		AttemptID:          job.AttemptID,
		Archive:            receipt.Object,
		FrameCount:         archive.FrameCount,
		ArchiveSizeBytes:   receipt.SizeBytes,
		ProcessingDuration: archive.ProcessingDuration,
	}
	if err := result.Validate(); err != nil {
		return outcome, fmt.Errorf("%w: %w", ErrInvalidPortResult, err)
	}
	if receipt.Object.Bucket != assignment.Destination.Bucket || receipt.Object.Key != assignment.Destination.Key ||
		(assignment.Destination.VersionID != "" && receipt.Object.VersionID != assignment.Destination.VersionID) {
		return outcome, fmt.Errorf("recibo não corresponde ao destino autorizado: %w", ErrInvalidPortResult)
	}
	outcome.Result = &result
	if err := ctx.Err(); err != nil {
		return outcome, err
	}
	if err := e.coordinator.Complete(ctx, result); err != nil {
		return outcome, fmt.Errorf("confirmar resultado: %w", err)
	}
	outcome.Confirmed = true
	return outcome, nil
}

func closeResource(resource io.Closer, operation string, executionErr *error) {
	if err := resource.Close(); err != nil {
		*executionErr = errors.Join(*executionErr, fmt.Errorf("%s: %w", operation, err))
	}
}
