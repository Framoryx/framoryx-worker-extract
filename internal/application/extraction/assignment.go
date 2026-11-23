package extraction

import (
	"fmt"
	"slices"

	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

func (a Assignment) completedOutcome(request ExecutionRequest) (ExecutionOutcome, error) {
	result := *a.Completed
	if a.Job != nil || result.JobID != request.JobID {
		return ExecutionOutcome{}, ErrInvalidAssignment
	}
	if err := result.Validate(); err != nil {
		return ExecutionOutcome{}, fmt.Errorf("%w: %w", ErrInvalidAssignment, err)
	}
	return ExecutionOutcome{Result: &result, Confirmed: true}, nil
}

func (a Assignment) authorizedJob(request ExecutionRequest) (domain.ExtractionJob, error) {
	if a.Job == nil || a.Job.ID != request.JobID {
		return domain.ExtractionJob{}, ErrInvalidAssignment
	}
	job := *a.Job
	job.Options.Segments = slices.Clone(job.Options.Segments)
	if err := job.Validate(); err != nil {
		return domain.ExtractionJob{}, fmt.Errorf("%w: %w", ErrInvalidAssignment, err)
	}
	if err := a.validateDestination(job.Source.Object); err != nil {
		return domain.ExtractionJob{}, err
	}
	return job, nil
}

func (a Assignment) validateDestination(source domain.ObjectReference) error {
	if err := a.Destination.Validate(); err != nil {
		return fmt.Errorf("destino autorizado: %w: %w", ErrInvalidAssignment, err)
	}
	if sameLocation(a.Destination, source) {
		return fmt.Errorf("destino do arquivo coincide com o vídeo original: %w", ErrInvalidAssignment)
	}
	return nil
}

func sameLocation(left, right domain.ObjectReference) bool {
	return left.Bucket == right.Bucket && left.Key == right.Key
}
