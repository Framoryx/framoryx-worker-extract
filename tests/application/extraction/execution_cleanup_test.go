package extraction_test

import (
	"errors"
	"testing"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
)

// TestExecutionCompletionAndCleanup preserves accepted results and all failure causes.
func TestExecutionCompletionAndCleanup(t *testing.T) {
	operationErr := errors.New("falha da operação simulada")
	sourceCloseErr := errors.New("falha ao fechar origem")
	archiveCloseErr := errors.New("falha ao fechar arquivo")
	runExecutionCases(t, map[string]executionCase{
		"resposta de confirmação perdida": {
			configure:  func(s *executionScenario) { s.completeErr = operationErr },
			wantErrors: []error{operationErr}, wantResult: true, wantEvents: completedEvents(),
		},
		"tentativa obsoleta na conclusão": {
			configure:  func(s *executionScenario) { s.completeErr = application.ErrAttemptObsolete },
			wantErrors: []error{application.ErrAttemptObsolete}, wantResult: true, wantEvents: completedEvents(),
		},
		"falha de confirmação e limpeza preservadas": {
			configure:  func(s *executionScenario) { s.completeErr, s.archiveCloseErr = operationErr, archiveCloseErr },
			wantErrors: []error{operationErr, archiveCloseErr}, wantResult: true, wantEvents: completedEvents(),
		},
		"limpeza falha após confirmação": {
			configure:  func(s *executionScenario) { s.archiveCloseErr = archiveCloseErr },
			wantErrors: []error{archiveCloseErr}, wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"ambas limpezas falham após confirmação": {
			configure:  func(s *executionScenario) { s.sourceCloseErr, s.archiveCloseErr = sourceCloseErr, archiveCloseErr },
			wantErrors: []error{sourceCloseErr, archiveCloseErr}, wantResult: true, wantConfirmed: true, wantEvents: completedEvents(),
		},
		"falha principal e limpeza preservadas": {
			configure:  func(s *executionScenario) { s.processErr, s.sourceCloseErr = operationErr, sourceCloseErr },
			wantErrors: []error{operationErr, sourceCloseErr}, wantEvents: processedEvents(),
		},
	})
}
