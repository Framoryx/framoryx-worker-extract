package extraction

// Executor orchestrates an extraction through infrastructure-independent ports.
type Executor struct {
	coordinator Coordinator
	source      SourceStorage
	processor   ArchiveProcessor
	archives    ArchiveStorage
}
