package extraction

import "time"

// ExtractionOptions defines the mode, sampling interval, required segment order and format.
type ExtractionOptions struct {
	Mode          ExtractionMode
	Segments      []VideoSegment
	FrameInterval time.Duration
	ImageFormat   ImageFormat
}
