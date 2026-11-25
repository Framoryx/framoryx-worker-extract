package extraction

import "strings"

func (r ExecutionRequest) validate() error {
	if strings.TrimSpace(r.JobID) == "" || strings.TrimSpace(r.AcquisitionKey) == "" {
		return ErrInvalidRequest
	}
	return nil
}
