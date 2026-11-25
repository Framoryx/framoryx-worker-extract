package extraction

import (
	"io"
	"time"
)

// ProcessedArchive owns the archive stream and its extraction measurements.
type ProcessedArchive struct {
	Body               io.ReadCloser
	FrameCount         int64
	ProcessingDuration time.Duration
}
