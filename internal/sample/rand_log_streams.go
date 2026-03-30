package sample

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"golang.org/x/sync/errgroup"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
)

// GetLogEvents has the lowest TPS - 10, and it's a bottleneck.
// See limits of other APIs here: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const concurrencyLimit = 10

func SampleByRandLogStreams(
	ctx context.Context,
	client *cloudwatchlogs.Client,
	cutoff int64,
	srcGroups []string,
	dstGroup string,
	randLogStreamsNumber int,
) {
	var g errgroup.Group
	g.SetLimit(concurrencyLimit)

	var processedLogStreams atomic.Int64

	for _, srcGroup := range srcGroups {
		g.Go(func() error {
			processed, err := processLogGroup(ctx, client, cutoff, srcGroup, dstGroup, randLogStreamsNumber)
			if err != nil {
				fmt.Printf("error processing log group %s: %v\n", srcGroup, err)
				return nil
			}
			processedLogStreams.Add(processed)
			return nil
		})
	}

	_ = g.Wait()
	log.Printf("processed %d log streams\n", processedLogStreams.Load())
}

func processLogGroup(
	ctx context.Context,
	client *cloudwatchlogs.Client,
	cutoff int64,
	srcGroup, dstGroup string,
	randLogStreamsNumber int,
) (processed int64, err error) {
	allStreams, err := cloudwatchlogs.DescribeLogStreamsUntilCutoff(ctx, client, srcGroup, cutoff)
	if err != nil {
		return processed, fmt.Errorf("describe log streams: %w", err)
	}
	if len(allStreams) == 0 {
		fmt.Println("no log streams found")
		return processed, nil
	}
	fmt.Printf("number of log streams: %d\n", len(allStreams))

	randStreams := pickRandomLogStreams(allStreams, randLogStreamsNumber)
	randStreamNames := make([]string, len(randStreams))
	for i, s := range randStreams {
		randStreamNames[i] = aws.ToString(s.LogStreamName)
	}
	fmt.Printf("randomly selected streams: %v\n", randStreamNames)

	for _, srcStreamName := range randStreamNames {
		// logGroupName/streamName/year/month/day/hour/minutes - almost the same format as CloudWatch Data Protection uses
		dstStreamName := fmt.Sprintf(
			"%s/%s/%s",
			srcGroup,
			srcStreamName,
			time.Now().UTC().Format("2006/01/02/15/04"),
		)
		fmt.Println("destination stream name:", dstStreamName)

		err = cloudwatchlogs.CopyLogStream(
			ctx,
			client,
			srcGroup,
			srcStreamName,
			dstGroup,
			dstStreamName,
		)
		if err != nil {
			fmt.Printf("error copying log stream %s: %v\n", srcStreamName, err)
			continue
		}
		processed++
	}

	return processed, nil
}

func pickRandomLogStreams(logStreams []types.LogStream, n int) []types.LogStream {
	total := len(logStreams)
	if total <= n {
		return logStreams
	}

	indices := rand.Perm(total)[:n]
	result := make([]types.LogStream, n)
	for i, idx := range indices {
		result[i] = logStreams[idx]
	}

	return result
}
