package extraction_test

import (
	"errors"
	"slices"
	"testing"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
)

func runExecutionCases(t *testing.T, cases map[string]executionCase) {
	t.Helper()
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			scenario := newScenario(t)
			if test.configure != nil {
				test.configure(scenario)
			}
			executeScenario(t, scenario, test)
		})
	}
}

func executeScenario(t *testing.T, scenario *executionScenario, test executionCase) application.ExecutionOutcome {
	t.Helper()
	executor, err := application.NewExecutor(scenario, scenario, scenario, scenario)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := executor.Execute(scenario.ctx, scenario.request)
	assertExecutionErrors(t, err, test.wantErrors)
	if (outcome.Result != nil) != test.wantResult || outcome.Confirmed != test.wantConfirmed {
		t.Errorf("estado do resultado inesperado: %+v", outcome)
	}
	if !slices.Equal(scenario.events, test.wantEvents) {
		t.Errorf("operações = %v; esperado %v", scenario.events, test.wantEvents)
	}
	return outcome
}

func assertExecutionErrors(t *testing.T, err error, wanted []error) {
	t.Helper()
	if len(wanted) == 0 && err != nil {
		t.Fatalf("execução falhou: %v", err)
	}
	for _, cause := range wanted {
		if !errors.Is(err, cause) {
			t.Errorf("erro = %v; esperado reconhecer %v", err, cause)
		}
	}
}

func processedEvents() []string {
	return []string{"adquirir", "abrir", "processar", "fechar arquivo", "fechar origem"}
}

func uploadedEvents() []string {
	return []string{"adquirir", "abrir", "processar", "armazenar", "fechar arquivo", "fechar origem"}
}

func completedEvents() []string {
	return []string{"adquirir", "abrir", "processar", "armazenar", "confirmar", "fechar arquivo", "fechar origem"}
}
