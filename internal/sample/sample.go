package sample

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	LogGroupNamePattern string
	DstGroup            string
	Type                string
	Cutoff              int64
}

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

	// TODO: process each log group using a specific type
	for _, srcGroup := range srcGroups {
		g.Go(func() error {
			if err := ProcessLogGroup(ctx, client, cfg.Cutoff, srcGroup, cfg.DstGroup); err != nil {
				fmt.Printf("error processing log group %s: %v\n", srcGroup, err)
			}
			return nil
		})
	}
	
	return g.Wait()
}

func ProcessLogGroup(ctx context.Context, client *cloudwatchlogs.Client, cutoff int64, srcGroup, dstGroup string) error {
	logStreams, err := cloudwatchlogs.DescribeLogStreamsUntilCutoff(ctx, client, srcGroup, cutoff)
	if err != nil {
		return fmt.Errorf("describe log streams: %w", err)
	}
	if len(logStreams) == 0 {
		fmt.Println("no log streams found")
		return nil
	}

	randIndex := rand.IntN(len(logStreams))
	// TODO: handle the case when the stream was already processed: if the destination stream
	// already exists, we should pick another one
	srcStream := logStreams[randIndex]

	fmt.Printf("number of log streams: %d\n", len(logStreams))
	fmt.Printf("randomly selected stream: %s\n", *srcStream.LogStreamName)

	// logGroupName/year/month/day/hour/minutes - almost the same format as CloudWatch Data Protection uses
	dstStreamName := fmt.Sprintf("%s/%s", srcGroup, time.Now().UTC().Format("2006/01/02/15/04"))
	fmt.Println("destination stream name:", dstStreamName)

	err = cloudwatchlogs.CopyLogStream(
		ctx,
		client,
		srcGroup,
		aws.ToString(srcStream.LogStreamName),
		dstGroup,
		dstStreamName,
	)
	if err != nil {
		return fmt.Errorf("copy log stream: %w", err)
	}

	return nil
}
