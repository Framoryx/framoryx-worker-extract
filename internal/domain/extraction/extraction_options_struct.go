package extraction

import "time"

// ExtractionOptions specifies frame spacing and image format.
type ExtractionOptions struct {
	FrameInterval time.Duration
	ImageFormat   ImageFormat
}
