package extraction_test

import (
	"testing"

	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestExecutionSuccess checks valid selections, destinations and preserved identities.
func TestExecutionSuccess(t *testing.T) {
	runExecutionCases(t, map[string]executionCase{
		"sucesso": {wantResult: true, wantConfirmed: true, wantEvents: completedEvents()},
		"todos os frames": {
			configure: func(s *executionScenario) {
				s.assignment.Job.Options.Mode, s.assignment.Job.Options.FrameInterval = domain.ExtractionModeAllFrames, 0
				s.expectedOptions.Mode, s.expectedOptions.FrameInterval = domain.ExtractionModeAllFrames, 0
			},
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"sem versão de destino": {
			configure:  func(s *executionScenario) { s.receipt.Object.VersionID = "" },
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"versão autorizada preservada": {
			configure:  func(s *executionScenario) { s.assignment.Destination.VersionID = s.receipt.Object.VersionID },
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"duração zero": {
			configure:  func(s *executionScenario) { s.duration = 0 },
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"destino no mesmo bucket com chave distinta": {
			configure: func(s *executionScenario) {
				s.assignment.Destination.Bucket = s.assignment.Job.Source.Object.Bucket
				s.receipt.Object.Bucket = s.assignment.Destination.Bucket
			},
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"destino com mesma chave em outro bucket": {
			configure: func(s *executionScenario) {
				s.assignment.Destination.Key = s.assignment.Job.Source.Object.Key
				s.receipt.Object.Key = s.assignment.Destination.Key
			},
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"snapshot da seleção": {
			configure:  func(s *executionScenario) { s.mutateAssigned = true },
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"identidade opaca preservada": {
			configure: func(s *executionScenario) {
				s.request.JobID, s.request.AcquisitionKey = " job customizado ", " aquisição customizada "
				s.assignment.Job.ID = s.request.JobID
			},
			wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
	})
}
