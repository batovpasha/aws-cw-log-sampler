package sample

import (
	"context"
	"fmt"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	LogGroupNamePattern string
	DstGroup            string
	Type                string
	Cutoff              int64
}

const (
	TypeRandLogStreams = "rand-log-streams"
)

// GetLogEvents has the lowest TPS - 10, and it's a bottleneck.
// See limits of other APIs here: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const concurrencyLimit = 10

func Sample(ctx context.Context, client *cloudwatchlogs.Client, cfg *Config) error {
	srcGroups, err := cloudwatchlogs.ListLogGroupNames(ctx, client, cfg.LogGroupNamePattern)
	if err != nil {
		return fmt.Errorf("list log groups: %w", err)
	}

	var g errgroup.Group
	g.SetLimit(concurrencyLimit)

	for _, srcGroup := range srcGroups {
		g.Go(func() error {
			var err error

			switch cfg.Type {
			case TypeRandLogStreams:
				err = SampleByRandLogStreams(ctx, client, cfg.Cutoff, srcGroup, cfg.DstGroup)
			default:
				err = fmt.Errorf("unsupported sampling type: %s", cfg.Type)
			}
			if err != nil {
				fmt.Printf("error processing log group %s: %v\n", srcGroup, err)
			}

			return nil
		})
	}

	return g.Wait()
}
