package sample

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"golang.org/x/sync/errgroup"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cwlogs"
)

// GetLogEvents has the lowest TPS - 10, and it's a bottleneck.
// See limits of other APIs here: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const concurrencyLimit = 10

func SampleByRandLogStreams(
	ctx context.Context,
	client *cwlogs.Client,
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
				slog.WarnContext(ctx, "error processing log group", "log_group", srcGroup, "error", err)
				return nil
			}
			processedLogStreams.Add(processed)
			return nil
		})
	}

	_ = g.Wait()
	slog.InfoContext(ctx, "complete log stream processing", "number", processedLogStreams.Load())
}

func processLogGroup(
	ctx context.Context,
	client *cwlogs.Client,
	cutoff int64,
	srcGroup, dstGroup string,
	randLogStreamsNumber int,
) (processed int64, err error) {
	allStreams, err := cwlogs.DescribeLogStreamsUntilCutoff(ctx, client, srcGroup, cutoff)
	if err != nil {
		return processed, fmt.Errorf("describe log streams: %w", err)
	}
	if len(allStreams) == 0 {
		slog.InfoContext(ctx, "no log streams found")
		return processed, nil
	}
	slog.InfoContext(ctx, "list log streams", "number", len(allStreams))

	randStreams := pickRandomLogStreams(allStreams, randLogStreamsNumber)
	randStreamNames := make([]string, len(randStreams))
	for i, s := range randStreams {
		randStreamNames[i] = aws.ToString(s.LogStreamName)
	}
	slog.InfoContext(ctx, "pick random log streams", "number", len(randStreamNames))

	for _, srcStreamName := range randStreamNames {
		// logGroupName/streamName/year/month/day/hour/minutes - almost the same format as CloudWatch Data Protection uses
		dstStreamName := fmt.Sprintf(
			"%s/%s/%s",
			srcGroup,
			srcStreamName,
			time.Now().UTC().Format("2006/01/02/15/04"),
		)

		slog.InfoContext(ctx, "copy log streams", "src_stream", srcStreamName, "dst_stream", dstStreamName)
		err = cwlogs.CopyLogStream(
			ctx,
			client,
			srcGroup,
			srcStreamName,
			dstGroup,
			dstStreamName,
		)
		if err != nil {
			slog.WarnContext(ctx, "error copying log stream", "error", err)
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
