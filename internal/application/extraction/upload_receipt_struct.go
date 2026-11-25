package extraction

import domain "github.com/Framoryx/framoryx-worker-extract/internal/domain/extraction"

// UploadReceipt identifies the stored archive and its confirmed size.
type UploadReceipt struct {
	Object    domain.ObjectReference
	SizeBytes int64
}
