package extraction

// ExtractionJob describes one identified attempt to extract video frames.
type ExtractionJob struct {
	ID        string
	AttemptID string
	Source    VideoSource
	Options   ExtractionOptions
}
