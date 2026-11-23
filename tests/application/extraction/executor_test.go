package extraction_test

import (
	"context"
	"errors"
	"testing"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestExecutionCancellation checks cancellation at every side-effect boundary.
func TestExecutionCancellation(t *testing.T) {
	cases := map[string]executionCase{
		"adquirir":  {wantEvents: []string{"adquirir"}},
		"abrir":     {wantEvents: []string{"adquirir", "abrir", "fechar origem"}},
		"processar": {wantEvents: processedEvents()},
		"armazenar": {wantResult: true, wantEvents: uploadedEvents()},
		"confirmar": {wantResult: true, wantConfirmed: true, wantEvents: completedEvents()},
	}
	for stage, test := range cases {
		t.Run(stage, func(t *testing.T) {
			scenario := newScenario(t)
			scenario.cancelAt = stage
			if !test.wantConfirmed {
				test.wantErrors = []error{context.Canceled}
			}
			outcome := executeScenario(t, scenario, test)
			if outcome.Result != nil && *outcome.Result != completedResult() {
				t.Errorf("metadados perdidos após cancelamento: %+v", outcome.Result)
			}
		})
	}
}

// TestCompletedAssignment checks duplicate delivery without repeating processing or confirmation.
func TestCompletedAssignment(t *testing.T) {
	operationErr := errors.New("falha ao obter resultado concluído")
	cases := map[string]executionCase{
		"resultado aceito": {wantResult: true, wantConfirmed: true},
		"resultado aceito antes de cancelar": {
			configure:  func(s *executionScenario) { s.cancelAt = "adquirir" },
			wantResult: true, wantConfirmed: true,
		},
		"resultado de outro job": {
			configure:  func(s *executionScenario) { s.assignment.Completed.JobID = "outro-job" },
			wantErrors: []error{application.ErrInvalidAssignment},
		},
		"resultado inválido": {
			configure:  func(s *executionScenario) { s.assignment.Completed.FrameCount = 0 },
			wantErrors: []error{application.ErrInvalidAssignment, domain.ErrInvalidResult},
		},
		"atribuição ambígua": {
			configure:  func(s *executionScenario) { s.assignment.Job = &domain.ExtractionJob{ID: "job-1"} },
			wantErrors: []error{application.ErrInvalidAssignment},
		},
		"erro prevalece sobre resultado retornado": {
			configure:  func(s *executionScenario) { s.acquireErr = operationErr },
			wantErrors: []error{operationErr},
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			scenario := newScenario(t)
			result := completedResult()
			scenario.assignment = application.Assignment{Completed: &result}
			if test.configure != nil {
				test.configure(scenario)
			}
			test.wantEvents = []string{"adquirir"}
			outcome := executeScenario(t, scenario, test)
			if outcome.Result != nil && outcome.Result == scenario.assignment.Completed {
				t.Fatal("resultado retornado compartilha a referência da dependência")
			}
			if outcome.Result != nil && *outcome.Result != result {
				t.Errorf("conteúdo do resultado anterior alterado: %+v", outcome.Result)
			}
		})
	}
}

// TestExecutorDependencies rejects absent ports and safely rejects an uninitialized executor.
func TestExecutorDependencies(t *testing.T) {
	factories := map[string]func(*executionScenario) (*application.Executor, error){
		"coordenação": func(s *executionScenario) (*application.Executor, error) {
			return application.NewExecutor(nil, s, s, s)
		},
		"origem": func(s *executionScenario) (*application.Executor, error) {
			return application.NewExecutor(s, nil, s, s)
		},
		"processamento": func(s *executionScenario) (*application.Executor, error) {
			return application.NewExecutor(s, s, nil, s)
		},
		"arquivo": func(s *executionScenario) (*application.Executor, error) {
			return application.NewExecutor(s, s, s, nil)
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			executor, err := factory(newScenario(t))
			if executor != nil || !errors.Is(err, application.ErrInvalidDependencies) {
				t.Fatalf("dependência ausente aceita: %v, %v", executor, err)
			}
		})
	}
	for _, executor := range []*application.Executor{nil, new(application.Executor)} {
		if _, err := executor.Execute(context.Background(), application.ExecutionRequest{JobID: "job-1", AcquisitionKey: "acquire-1"}); !errors.Is(err, application.ErrInvalidDependencies) {
			t.Fatalf("executor não inicializado aceito: %v", err)
		}
	}
}
