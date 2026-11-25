package extraction

import domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"

// Assignment contains either an authorized job or an already accepted result.
type Assignment struct {
	Job         *domain.ExtractionJob
	Destination domain.ObjectReference
	Completed   *domain.ExtractionResult
}
