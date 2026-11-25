package extraction

import domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"

// ExecutionOutcome retains a validated upload result and whether completion was accepted.
type ExecutionOutcome struct {
	Result    *domain.ExtractionResult
	Confirmed bool
}
