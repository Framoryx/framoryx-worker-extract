package logging

import "log/slog"

type handlerStep struct {
	group string
	attrs []slog.Attr
}
