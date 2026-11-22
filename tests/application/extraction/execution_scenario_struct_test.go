package extraction_test

import (
	"context"
	"testing"
	"time"

	application "github.com/Framoryx/framoryx-worker-extract/internal/application/extraction"
	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

type executionScenario struct {
	t               *testing.T
	ctx             context.Context
	cancel          context.CancelFunc
	cancelAt        string
	request         application.ExecutionRequest
	assignment      application.Assignment
	receipt         application.UploadReceipt
	expectedOptions domain.ExtractionOptions
	events          []string
	acquireErr      error
	openErr         error
	processErr      error
	storeErr        error
	completeErr     error
	sourceCloseErr  error
	archiveCloseErr error
	sourceNil       bool
	archiveNil      bool
	mutateAssigned  bool
	frameCount      int64
	duration        time.Duration
}
