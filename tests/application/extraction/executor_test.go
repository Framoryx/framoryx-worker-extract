package extraction_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// TestExecutionOrchestration checks ordering, validation, ambiguity and owned resource cleanup.
func TestExecutionOrchestration(t *testing.T) {
	operationErr := errors.New("falha da operação simulada")
	sourceCloseErr := errors.New("falha ao fechar origem")
	archiveCloseErr := errors.New("falha ao fechar arquivo")
	complete := []string{"adquirir", "abrir", "processar", "armazenar", "confirmar", "fechar arquivo", "fechar origem"}
	uploaded := []string{"adquirir", "abrir", "processar", "armazenar", "fechar arquivo", "fechar origem"}
	processed := []string{"adquirir", "abrir", "processar", "fechar arquivo", "fechar origem"}
	cases := map[string]executionCase{
		"sucesso": {wantResult: true, wantConfirmed: true, wantEvents: complete},
		"todos os frames": {
			configure: func(s *executionScenario) {
				s.assignment.Job.Options.Mode, s.assignment.Job.Options.FrameInterval = domain.ExtractionModeAllFrames, 0
				s.expectedOptions.Mode, s.expectedOptions.FrameInterval = domain.ExtractionModeAllFrames, 0
			},
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"sem versão de destino": {
			configure:  func(s *executionScenario) { s.receipt.Object.VersionID = "" },
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"versão autorizada preservada": {
			configure:  func(s *executionScenario) { s.assignment.Destination.VersionID = s.receipt.Object.VersionID },
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"duração zero": {
			configure:  func(s *executionScenario) { s.duration = 0 },
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"destino no mesmo bucket com chave distinta": {
			configure: func(s *executionScenario) {
				s.assignment.Destination.Bucket = s.assignment.Job.Source.Object.Bucket
				s.receipt.Object.Bucket = s.assignment.Destination.Bucket
			},
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"destino com mesma chave em outro bucket": {
			configure: func(s *executionScenario) {
				s.assignment.Destination.Key = s.assignment.Job.Source.Object.Key
				s.receipt.Object.Key = s.assignment.Destination.Key
			},
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"snapshot da seleção": {
			configure:  func(s *executionScenario) { s.mutateAssigned = true },
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"identidade opaca preservada": {
			configure: func(s *executionScenario) {
				s.request.JobID, s.request.AcquisitionKey = " job customizado ", " aquisição customizada "
				s.assignment.Job.ID = s.request.JobID
			},
			wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"job vazio":                    {configure: func(s *executionScenario) { s.request.JobID = "" }, wantErrors: []error{application.ErrInvalidRequest}},
		"job em branco":                {configure: func(s *executionScenario) { s.request.JobID = " \t" }, wantErrors: []error{application.ErrInvalidRequest}},
		"aquisição vazia":              {configure: func(s *executionScenario) { s.request.AcquisitionKey = "" }, wantErrors: []error{application.ErrInvalidRequest}},
		"aquisição em branco":          {configure: func(s *executionScenario) { s.request.AcquisitionKey = " \t" }, wantErrors: []error{application.ErrInvalidRequest}},
		"contexto ausente":             {configure: func(s *executionScenario) { s.ctx = nil }, wantErrors: []error{application.ErrInvalidRequest}},
		"cancelado antes da aquisição": {configure: func(s *executionScenario) { s.cancel() }, wantErrors: []error{context.Canceled}},
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
			wantErrors: []error{operationErr}, wantEvents: processed,
		},
		"arquivo gerado ausente": {
			configure:  func(s *executionScenario) { s.archiveNil = true },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: []string{"adquirir", "abrir", "processar", "fechar origem"},
		},
		"sem frames antes do upload": {
			configure:  func(s *executionScenario) { s.frameCount = 0 },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: processed,
		},
		"frames negativos antes do upload": {
			configure:  func(s *executionScenario) { s.frameCount = -1 },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: processed,
		},
		"duração inválida antes do upload": {
			configure:  func(s *executionScenario) { s.duration = -time.Second },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: processed,
		},
		"armazenamento falha": {
			configure:  func(s *executionScenario) { s.storeErr = operationErr },
			wantErrors: []error{operationErr}, wantEvents: uploaded,
		},
		"recibo sem referência": {
			configure:  func(s *executionScenario) { s.receipt.Object = domain.ObjectReference{} },
			wantErrors: []error{application.ErrInvalidPortResult, domain.ErrInvalidObjectReference}, wantEvents: uploaded,
		},
		"recibo de outro bucket": {
			configure:  func(s *executionScenario) { s.receipt.Object.Bucket = "outro-bucket" },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: uploaded,
		},
		"recibo de outra chave": {
			configure:  func(s *executionScenario) { s.receipt.Object.Key = "outro-objeto" },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: uploaded,
		},
		"recibo de outra versão autorizada": {
			configure:  func(s *executionScenario) { s.assignment.Destination.VersionID = "versão-esperada" },
			wantErrors: []error{application.ErrInvalidPortResult}, wantEvents: uploaded,
		},
		"recibo com tamanho zero": {
			configure:  func(s *executionScenario) { s.receipt.SizeBytes = 0 },
			wantErrors: []error{application.ErrInvalidPortResult, domain.ErrInvalidResult}, wantEvents: uploaded,
		},
		"recibo com tamanho negativo": {
			configure:  func(s *executionScenario) { s.receipt.SizeBytes = -1 },
			wantErrors: []error{application.ErrInvalidPortResult, domain.ErrInvalidResult}, wantEvents: uploaded,
		},
		"resposta de confirmação perdida": {
			configure:  func(s *executionScenario) { s.completeErr = operationErr },
			wantErrors: []error{operationErr}, wantResult: true, wantEvents: complete,
		},
		"tentativa obsoleta na conclusão": {
			configure:  func(s *executionScenario) { s.completeErr = application.ErrAttemptObsolete },
			wantErrors: []error{application.ErrAttemptObsolete}, wantResult: true, wantEvents: complete,
		},
		"falha de confirmação e limpeza preservadas": {
			configure:  func(s *executionScenario) { s.completeErr, s.archiveCloseErr = operationErr, archiveCloseErr },
			wantErrors: []error{operationErr, archiveCloseErr}, wantResult: true, wantEvents: complete,
		},
		"limpeza falha após confirmação": {
			configure:  func(s *executionScenario) { s.archiveCloseErr = archiveCloseErr },
			wantErrors: []error{archiveCloseErr}, wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"ambas limpezas falham após confirmação": {
			configure:  func(s *executionScenario) { s.sourceCloseErr, s.archiveCloseErr = sourceCloseErr, archiveCloseErr },
			wantErrors: []error{sourceCloseErr, archiveCloseErr}, wantResult: true, wantConfirmed: true, wantEvents: complete,
		},
		"falha principal e limpeza preservadas": {
			configure:  func(s *executionScenario) { s.processErr, s.sourceCloseErr = operationErr, sourceCloseErr },
			wantErrors: []error{operationErr, sourceCloseErr}, wantEvents: processed,
		},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			scenario := newScenario(t)
			if test.configure != nil {
				test.configure(scenario)
			}
			executor, err := application.NewExecutor(scenario, scenario, scenario, scenario)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := executor.Execute(scenario.ctx, scenario.request)
			if len(test.wantErrors) == 0 && err != nil {
				t.Fatalf("execução falhou: %v", err)
			}
			for _, wanted := range test.wantErrors {
				if !errors.Is(err, wanted) {
					t.Errorf("erro = %v; esperado reconhecer %v", err, wanted)
				}
			}
			if (outcome.Result != nil) != test.wantResult || outcome.Confirmed != test.wantConfirmed {
				t.Errorf("estado do resultado inesperado: %+v", outcome)
			}
			if !slices.Equal(scenario.events, test.wantEvents) {
				t.Errorf("operações = %v; esperado %v", scenario.events, test.wantEvents)
			}
		})
	}
}

// TestExecutionCancellation checks cancellation at every side-effect boundary.
func TestExecutionCancellation(t *testing.T) {
	cases := map[string]executionCase{
		"adquirir":  {wantEvents: []string{"adquirir"}},
		"abrir":     {wantEvents: []string{"adquirir", "abrir", "fechar origem"}},
		"processar": {wantEvents: []string{"adquirir", "abrir", "processar", "fechar arquivo", "fechar origem"}},
		"armazenar": {wantResult: true, wantEvents: []string{"adquirir", "abrir", "processar", "armazenar", "fechar arquivo", "fechar origem"}},
		"confirmar": {wantResult: true, wantConfirmed: true, wantEvents: []string{"adquirir", "abrir", "processar", "armazenar", "confirmar", "fechar arquivo", "fechar origem"}},
	}
	for stage, test := range cases {
		t.Run(stage, func(t *testing.T) {
			scenario := newScenario(t)
			scenario.cancelAt = stage
			executor, err := application.NewExecutor(scenario, scenario, scenario, scenario)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := executor.Execute(scenario.ctx, scenario.request)
			if test.wantConfirmed {
				if err != nil {
					t.Fatalf("confirmação aceita foi convertida em falha: %v", err)
				}
			} else if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelamento não propagado: %v", err)
			}
			if (outcome.Result != nil) != test.wantResult || outcome.Confirmed != test.wantConfirmed || !slices.Equal(scenario.events, test.wantEvents) {
				t.Fatalf("execução após cancelamento: %+v; operações %v", outcome, scenario.events)
			}
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
			executor, err := application.NewExecutor(scenario, scenario, scenario, scenario)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := executor.Execute(scenario.ctx, scenario.request)
			if len(test.wantErrors) == 0 && err != nil {
				t.Fatal(err)
			}
			for _, wanted := range test.wantErrors {
				if !errors.Is(err, wanted) {
					t.Errorf("erro = %v; esperado reconhecer %v", err, wanted)
				}
			}
			if (outcome.Result != nil) != test.wantResult || outcome.Confirmed != test.wantConfirmed || !slices.Equal(scenario.events, []string{"adquirir"}) {
				t.Fatalf("entrega duplicada produziu efeitos: %+v; %v", outcome, scenario.events)
			}
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
