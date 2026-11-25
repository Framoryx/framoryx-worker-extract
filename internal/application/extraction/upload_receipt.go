package extraction

import (
	"fmt"

	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

func (r UploadReceipt) result(job domain.ExtractionJob, archive ProcessedArchive, destination domain.ObjectReference) (*domain.ExtractionResult, error) {
	result := domain.ExtractionResult{
		JobID:              job.ID,
		VideoID:            job.Source.VideoID,
		AttemptID:          job.AttemptID,
		Archive:            r.Object,
		FrameCount:         archive.FrameCount,
		ArchiveSizeBytes:   r.SizeBytes,
		ProcessingDuration: archive.ProcessingDuration,
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPortResult, err)
	}
	if !r.matchesDestination(destination) {
		return nil, fmt.Errorf("recibo não corresponde ao destino autorizado: %w", ErrInvalidPortResult)
	}
	return &result, nil
}

func (r UploadReceipt) matchesDestination(destination domain.ObjectReference) bool {
	return sameLocation(r.Object, destination) && (destination.VersionID == "" || r.Object.VersionID == destination.VersionID)
}
