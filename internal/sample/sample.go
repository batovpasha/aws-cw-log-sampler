package sample

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cwlogs"
)

type Config struct {
	LogGroupNamePattern string
	DstGroup            string
	Type                string
	Cutoff              int64

	// rand-log-streams options
	RandLogStreamsNumber int
}

const (
	TypeRandLogStreams = "rand-log-streams"
)

func Sample(ctx context.Context, client *cwlogs.Client, cfg *Config) error {
	srcGroups, err := cwlogs.ListLogGroupNames(ctx, client, cfg.LogGroupNamePattern)
	if err != nil {
		return fmt.Errorf("list log groups: %w", err)
	}
	slog.InfoContext(ctx, "list log groups", "number", len(srcGroups))

	switch cfg.Type {
	case TypeRandLogStreams:
		SampleByRandLogStreams(ctx, client, cfg.Cutoff, srcGroups, cfg.DstGroup, cfg.RandLogStreamsNumber)
	}

	return nil
}
