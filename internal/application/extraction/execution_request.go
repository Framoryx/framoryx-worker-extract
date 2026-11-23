package extraction

import (
	"context"
	"strings"
)

func (r ExecutionRequest) validate(ctx context.Context) error {
	if ctx == nil || strings.TrimSpace(r.JobID) == "" || strings.TrimSpace(r.AcquisitionKey) == "" {
		return ErrInvalidRequest
	}
	return nil
}
