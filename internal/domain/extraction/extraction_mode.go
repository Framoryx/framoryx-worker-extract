package extraction

// ExtractionMode identifies how frames are selected within each segment.
type ExtractionMode string

// ExtractionModeSampling selects frames at an explicit interval.
const ExtractionModeSampling ExtractionMode = "sampling"

// ExtractionModeAllFrames selects every frame within each segment.
const ExtractionModeAllFrames ExtractionMode = "all_frames"
