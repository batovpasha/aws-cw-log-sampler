package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cli"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
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

func main() {
	fs := flag.CommandLine
	flags := cli.RegisterCommonFlags(fs)
	lookbackHours := fs.Int("lookbackHours", 24, "lookback hours")
	_ = fs.Parse(os.Args[1:])

	err := cli.ValidateCommonFlags(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *lookbackHours <= 0 {
		fmt.Fprintln(os.Stderr, "--lookbackHours should be an integer")
		os.Exit(1)
	}

	h := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(&contextHandler{Handler: h})
	slog.SetDefault(logger)
	traceID := time.Now().UTC().Format(time.RFC3339)
	ctx := context.WithValue(context.Background(), traceIDKey, traceID)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	cutoff := time.Now().Add(-time.Duration(*lookbackHours) * time.Hour).UnixMilli()

	err = sample.Sample(ctx, client, &sample.Config{
		LogGroupNamePattern:  flags.LogGroupNamePattern,
		DstGroup:             flags.DstGroup,
		Type:                 flags.Type,
		Cutoff:               cutoff,
		RandLogStreamsNumber: flags.RandLogStreamsNumber,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
