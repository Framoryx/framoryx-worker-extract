package extraction_test

type validationCase[T any] struct {
	change func(*T)
	want   error
}
