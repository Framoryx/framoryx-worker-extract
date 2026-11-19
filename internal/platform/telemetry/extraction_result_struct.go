package telemetry

type ExtractionResult struct {
	FrameCount       int64
	ArchiveSizeBytes int64
	Err              error
}
