package extraction

func (a ProcessedArchive) validate() error {
	if a.Body == nil || a.FrameCount <= 0 || a.ProcessingDuration < 0 {
		return ErrInvalidPortResult
	}
	return nil
}
