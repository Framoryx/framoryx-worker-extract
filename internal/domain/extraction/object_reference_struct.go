package extraction

// ObjectReference identifies a stored object with an optional version.
type ObjectReference struct {
	Bucket    string
	Key       string
	VersionID string
}
