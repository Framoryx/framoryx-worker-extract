package tracecontext

import (
	"context"

	"github.com/google/uuid"
)

type contextKey uint8

const identifiersKey contextKey = 0

func ValidID(value string) bool {
	if len(value) != 36 {
		return false
	}
	id, err := uuid.Parse(value)
	return err == nil && id.Version() == 4 && id.Variant() == uuid.RFC4122 && id.String() == value
}

func FromContext(ctx context.Context) Identifiers {
	ids, _ := ctx.Value(identifiersKey).(Identifiers)
	return ids
}

func NewAttempt(ctx context.Context) context.Context {
	ids := FromContext(ctx)
	if !ValidID(ids.CorrelationID) {
		ids.CorrelationID = uuid.NewString()
	}
	ids.RequestID = uuid.NewString()
	return context.WithValue(ctx, identifiersKey, ids)
}
