package sample

import (
	"context"
	"fmt"
	"log"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
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

// GetLogEvents has the lowest TPS - 10, and it's a bottleneck.
// See limits of other APIs here: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const concurrencyLimit = 10

func Sample(ctx context.Context, client *cloudwatchlogs.Client, cfg *Config) error {
	srcGroups, err := cloudwatchlogs.ListLogGroupNames(ctx, client, cfg.LogGroupNamePattern)
	if err != nil {
		return fmt.Errorf("list log groups: %w", err)
	}
	log.Println("number of log groups:", len(srcGroups))

	var g errgroup.Group
	g.SetLimit(concurrencyLimit)

	var mu sync.Mutex
	processedLogStreams := 0

	// TODO: Move log stream processing to rand_log_streams.go because concurrencyLimit and processedLogStreams logging
	// are internal details of the rand-log-streams sampling type
	for _, srcGroup := range srcGroups {
		g.Go(func() error {
			switch cfg.Type {
			case TypeRandLogStreams:
				processed, err := SampleByRandLogStreams(
					ctx,
					client,
					cfg.Cutoff,
					srcGroup,
					cfg.DstGroup,
					cfg.RandLogStreamsNumber,
				)
				if err != nil {
					fmt.Printf("error processing log group %s: %v\n", srcGroup, err)
					return nil
				}
				mu.Lock()
				processedLogStreams += processed
				mu.Unlock()
			}

			return nil
		})
	}

	_ = g.Wait()
	log.Printf("processed %d log streams\n", processedLogStreams)

	return nil
}
