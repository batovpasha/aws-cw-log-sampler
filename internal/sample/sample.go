package sample

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
)

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
