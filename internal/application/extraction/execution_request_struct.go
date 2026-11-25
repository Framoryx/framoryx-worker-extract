package extraction

// ExecutionRequest identifies a job and a stable acquisition operation.
type ExecutionRequest struct {
	JobID          string
	AcquisitionKey string
}
