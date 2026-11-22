package extraction

import "time"

// VideoSegment selects source video time from Start inclusive to End exclusive.
type VideoSegment struct {
	Start time.Duration
	End   time.Duration
}
