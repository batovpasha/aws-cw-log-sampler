package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type contextHandler struct {
	slog.Handler
}
type ctxKey string

const traceIDKey ctxKey = "trace_id"

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceId, ok := ctx.Value(traceIDKey).(string); ok {
		r.AddAttrs(slog.String(string(traceIDKey), traceId))
	}
	return h.Handler.Handle(ctx, r)
}

func New(version string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, nil)
	l := slog.New(&contextHandler{Handler: h}).With("version", version)
	slog.SetDefault(l)
	return l
}

func WithTraceID(ctx context.Context) context.Context {
	traceID := time.Now().UTC().Format(time.RFC3339)
	return context.WithValue(ctx, traceIDKey, traceID)
}
