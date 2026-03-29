package sample

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
)

func SampleByRandLogStreams(
	ctx context.Context,
	client *cloudwatchlogs.Client,
	cutoff int64,
	srcGroup, dstGroup string,
	randLogStreamsNumber int,
) (processed int, err error) {
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
