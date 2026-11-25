package extraction_test

import (
	"context"
	"errors"
	"testing"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestExecutionRequestValidation rejects invalid requests before acquiring work.
func TestExecutionRequestValidation(t *testing.T) {
	runExecutionCases(t, map[string]executionCase{
		"job vazio":                    {configure: func(s *executionScenario) { s.request.JobID = "" }, wantErrors: []error{application.ErrInvalidRequest}},
		"job em branco":                {configure: func(s *executionScenario) { s.request.JobID = " \t" }, wantErrors: []error{application.ErrInvalidRequest}},
		"aquisição vazia":              {configure: func(s *executionScenario) { s.request.AcquisitionKey = "" }, wantErrors: []error{application.ErrInvalidRequest}},
		"aquisição em branco":          {configure: func(s *executionScenario) { s.request.AcquisitionKey = " \t" }, wantErrors: []error{application.ErrInvalidRequest}},
		"contexto ausente":             {configure: func(s *executionScenario) { s.ctx = nil }, wantErrors: []error{application.ErrInvalidRequest}},
		"cancelado antes da aquisição": {configure: func(s *executionScenario) { s.cancel() }, wantErrors: []error{context.Canceled}},
	})
}

// TestExecutionAssignmentValidation rejects acquisition failures and invalid assignments.
func TestExecutionAssignmentValidation(t *testing.T) {
	operationErr := errors.New("falha da operação simulada")
	runExecutionCases(t, map[string]executionCase{
		"erro de aquisição": {
			configure:  func(s *executionScenario) { s.acquireErr = operationErr },
			wantErrors: []error{operationErr}, wantEvents: []string{"adquirir"},
		},
		"outra tentativa ativa": {
			configure:  func(s *executionScenario) { s.acquireErr = application.ErrAttemptUnavailable },
			wantErrors: []error{application.ErrAttemptUnavailable}, wantEvents: []string{"adquirir"},
		},
		"atribuição ausente": {
			configure:  func(s *executionScenario) { s.assignment.Job = nil },
			wantErrors: []error{application.ErrInvalidAssignment}, wantEvents: []string{"adquirir"},
		},
		"job de outra solicitação": {
			configure:  func(s *executionScenario) { s.assignment.Job.ID = "outro-job" },
			wantErrors: []error{application.ErrInvalidAssignment}, wantEvents: []string{"adquirir"},
		},
		"tentativa sem identidade": {
			configure:  func(s *executionScenario) { s.assignment.Job.AttemptID = "" },
			wantErrors: []error{application.ErrInvalidAssignment, domain.ErrInvalidID}, wantEvents: []string{"adquirir"},
		},
		"seleção inválida na atribuição": {
			configure:  func(s *executionScenario) { s.assignment.Job.Options.Segments = nil },
			wantErrors: []error{application.ErrInvalidAssignment, domain.ErrInvalidSegments}, wantEvents: []string{"adquirir"},
		},
		"destino inválido": {
			configure:  func(s *executionScenario) { s.assignment.Destination.Key = "" },
			wantErrors: []error{application.ErrInvalidAssignment, domain.ErrInvalidObjectReference}, wantEvents: []string{"adquirir"},
		},
		"destino sobrescreveria a origem": {
			configure:  func(s *executionScenario) { s.assignment.Destination = s.assignment.Job.Source.Object },
			wantErrors: []error{application.ErrInvalidAssignment}, wantEvents: []string{"adquirir"},
		},
		"destino coincide com origem em outra versão": {
			configure: func(s *executionScenario) {
				s.assignment.Destination = s.assignment.Job.Source.Object
				s.assignment.Destination.VersionID = "outra-versão"
			},
			wantErrors: []error{application.ErrInvalidAssignment}, wantEvents: []string{"adquirir"},
		},
	})
}
