package extraction

import "time"

// ExtractionResult describes the archive produced by an extraction attempt.
type ExtractionResult struct {
	JobID              string
	VideoID            string
	AttemptID          string
	Archive            ObjectReference
	FrameCount         int64
	ArchiveSizeBytes   int64
	ProcessingDuration time.Duration
}
