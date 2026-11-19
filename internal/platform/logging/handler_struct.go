package logging

import "log/slog"

type Handler struct {
	next    slog.Handler
	service string
	steps   []handlerStep
}
