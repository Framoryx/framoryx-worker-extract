package extraction

import (
	"context"
	"fmt"
)

func runStage[T any](ctx context.Context, name string, operation func(context.Context) (T, error)) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	result, err := operation(ctx)
	if err != nil {
		return result, fmt.Errorf("%s: %w", name, err)
	}
	return result, nil
}
