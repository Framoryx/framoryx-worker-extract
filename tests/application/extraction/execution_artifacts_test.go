package extraction_test

import (
	"errors"
	"testing"
	"time"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestExecutionProcessingFailures closes partial resources and rejects invalid archives.
func TestExecutionProcessingFailures(t *testing.T) {
	operationErr := errors.New("falha da operação simulada")
	runExecutionCases(t, map[string]executionCase{
		"origem ausente": {
			configure:  func(s *executionScenario) { s.sourceNil = true },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: []string{"adquirir", "abrir"},
		},
		"abertura falha com recurso parcial": {
			configure:  func(s *executionScenario) { s.openErr = operationErr },
			wantErrors: []error{operationErr}, wantEvents: []string{"adquirir", "abrir", "fechar origem"},
		},
		"processamento falha com arquivo parcial": {
			configure:  func(s *executionScenario) { s.processErr = operationErr },
			wantErrors: []error{operationErr}, wantEvents: processedEvents(),
		},
		"arquivo gerado ausente": {
			configure:  func(s *executionScenario) { s.archiveNil = true },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: []string{"adquirir", "abrir", "processar", "fechar origem"},
		},
		"sem frames antes do upload": {
			configure:  func(s *executionScenario) { s.frameCount = 0 },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: processedEvents(),
		},
		"frames negativos antes do upload": {
			configure:  func(s *executionScenario) { s.frameCount = -1 },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: processedEvents(),
		},
		"duração inválida antes do upload": {
			configure:  func(s *executionScenario) { s.duration = -time.Second },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: processedEvents(),
		},
	})
}

// TestExecutionUploadValidation rejects failed uploads and invalid receipts.
func TestExecutionUploadValidation(t *testing.T) {
	operationErr := errors.New("falha da operação simulada")
	runExecutionCases(t, map[string]executionCase{
		"armazenamento falha": {
			configure:  func(s *executionScenario) { s.storeErr = operationErr },
			wantErrors: []error{operationErr}, wantEvents: uploadedEvents(),
		},
		"recibo sem referência": {
			configure:  func(s *executionScenario) { s.receipt.Object = domain.ObjectReference{} },
			wantErrors: []error{application.ErrInvalidPortResult, domain.ErrInvalidObjectReference}, wantEvents: uploadedEvents(),
		},
		"recibo de outro bucket": {
			configure:  func(s *executionScenario) { s.receipt.Object.Bucket = "outro-bucket" },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: uploadedEvents(),
		},
		"recibo de outra chave": {
			configure:  func(s *executionScenario) { s.receipt.Object.Key = "outro-objeto" },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: uploadedEvents(),
		},
		"recibo de outra versão autorizada": {
			configure:  func(s *executionScenario) { s.assignment.Destination.VersionID = "versão-esperada" },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: uploadedEvents(),
		},
		"recibo com tamanho zero": {
			configure:  func(s *executionScenario) { s.receipt.SizeBytes = 0 },
			wantErrors: []error{application.ErrInvalidPortResult, domain.ErrInvalidResult}, wantEvents: uploadedEvents(),
		},
		"recibo com tamanho negativo": {
			configure:  func(s *executionScenario) { s.receipt.SizeBytes = -1 },
			wantErrors: []error{application.ErrInvalidPortResult, domain.ErrInvalidResult}, wantEvents: uploadedEvents(),
		},
	})
}
