package extraction_test

import "io"

type ownedReader struct {
	reader  io.Reader
	closeFn func() error
	closed  bool
}
