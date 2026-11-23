package extraction

import (
	"context"
	"errors"
	"fmt"
	"io"

	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// NewExecutor requires all four execution ports.
func NewExecutor(coordinator Coordinator, source SourceStorage, processor ArchiveProcessor, archives ArchiveStorage) (*Executor, error) {
	executor := &Executor{coordinator: coordinator, source: source, processor: processor, archives: archives}
	if !executor.ready() {
		return nil, ErrInvalidDependencies
	}
	return executor, nil
}

// Execute acquires, produces, uploads and confirms one result without retrying.
func (e *Executor) Execute(ctx context.Context, request ExecutionRequest) (ExecutionOutcome, error) {
	if !e.ready() {
		return ExecutionOutcome{}, ErrInvalidDependencies
	}
	if err := request.validate(ctx); err != nil {
		return ExecutionOutcome{}, err
	}
	assignment, err := runStage(ctx, "adquirir tentativa", func(ctx context.Context) (Assignment, error) {
		return e.coordinator.Acquire(ctx, request)
	})
	if err != nil {
		return ExecutionOutcome{}, err
	}
	if assignment.Completed != nil {
		return assignment.completedOutcome(request)
	}
	job, err := assignment.authorizedJob(request)
	if err != nil {
		return ExecutionOutcome{}, err
	}
	return e.runAssigned(ctx, job, assignment.Destination)
}

func (e *Executor) ready() bool {
	return e != nil && e.coordinator != nil && e.source != nil && e.processor != nil && e.archives != nil
}

func (e *Executor) runAssigned(ctx context.Context, job domain.ExtractionJob, destination domain.ObjectReference) (outcome ExecutionOutcome, err error) {
	source, err := runStage(ctx, "abrir vídeo de origem", func(ctx context.Context) (io.ReadCloser, error) {
		return e.source.Open(ctx, job.Source.Object)
	})
	if source != nil {
		defer closeResource(source, "fechar vídeo de origem", &err)
	}
	if err != nil {
		return outcome, err
	}
	if source == nil {
		return outcome, ErrInvalidPortResult
	}
	archive, err := runStage(ctx, "processar vídeo", func(ctx context.Context) (ProcessedArchive, error) {
		return e.processor.Process(ctx, source, job.Options)
	})
	if archive.Body != nil {
		defer closeResource(archive.Body, "fechar arquivo gerado", &err)
	}
	if err != nil {
		return outcome, err
	}
	if err := archive.validate(); err != nil {
		return outcome, err
	}
	receipt, err := runStage(ctx, "armazenar arquivo gerado", func(ctx context.Context) (UploadReceipt, error) {
		return e.archives.Store(ctx, destination, archive.Body)
	})
	if err != nil {
		return outcome, err
	}
	outcome.Result, err = receipt.result(job, archive, destination)
	if err != nil {
		return outcome, err
	}
	outcome.Confirmed, err = runStage(ctx, "confirmar resultado", func(ctx context.Context) (bool, error) {
		err := e.coordinator.Complete(ctx, *outcome.Result)
		return err == nil, err
	})
	return outcome, err
}

func closeResource(resource io.Closer, operation string, executionErr *error) {
	if err := resource.Close(); err != nil {
		*executionErr = errors.Join(*executionErr, fmt.Errorf("%s: %w", operation, err))
	}
}
