package extraction

import (
	"context"
	"io"

	domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"
)

// Coordinator authorizes attempts and accepts their results durably.
type Coordinator interface {
	// Acquire returns an assignment, a completed result or ErrAttemptUnavailable.
	Acquire(ctx context.Context, request ExecutionRequest) (Assignment, error)
	// Complete accepts a result durably or rejects it with ErrAttemptObsolete.
	Complete(ctx context.Context, result domain.ExtractionResult) error
}

// SourceStorage opens the fixed original and transfers ownership of its stream.
type SourceStorage interface {
	// Open returns a stream that the executor must close, including on partial errors.
	Open(ctx context.Context, reference domain.ObjectReference) (io.ReadCloser, error)
}

// ArchiveProcessor extracts frames and packages them without fixing a concrete tool.
type ArchiveProcessor interface {
	// Process returns an owned archive stream, including any stream left by a partial error.
	Process(ctx context.Context, source io.Reader, options domain.ExtractionOptions) (ProcessedArchive, error)
}

// ArchiveStorage writes an archive to its authorized destination without owning the reader.
type ArchiveStorage interface {
	// Store returns a receipt only after a completed upload to the requested destination.
	Store(ctx context.Context, destination domain.ObjectReference, body io.Reader) (UploadReceipt, error)
}
