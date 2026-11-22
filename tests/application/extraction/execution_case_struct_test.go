package extraction_test

type executionCase struct {
	configure     func(*executionScenario)
	wantErrors    []error
	wantResult    bool
	wantConfirmed bool
	wantEvents    []string
}
