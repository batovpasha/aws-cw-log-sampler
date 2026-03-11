package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func ProcessLogGroup(ctx context.Context, client *cloudwatchlogs.Client, srcGroup, dstGroup string) error {
	var nextToken *string
	var logStreams []types.LogStream

Outer:
	for {
		output, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: aws.String(srcGroup),
			OrderBy:      types.OrderByLastEventTime,
			Descending:   aws.Bool(true),
			NextToken:    nextToken,
			Limit:        aws.Int32(50),
		})
		if err != nil {
			return fmt.Errorf("describe log streams: %w", err)
		}
		if len(output.LogStreams) == 0 {
			fmt.Println("no streams found in", srcGroup)
			break
		}

		for _, stream := range output.LogStreams {
			// LastEventTimestamp is in epoch milliseconds
			if aws.ToInt64(stream.LastEventTimestamp) < cutoff {
				// Streams are sorted descending by last event time,
				// so once we pass the cutoff, we're done.
				break Outer
			}

			logStreams = append(logStreams, stream)
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}

	randIndex := rand.IntN(len(logStreams))
	srcStream := logStreams[randIndex]

	fmt.Printf("number of log streams: %d\n", len(logStreams))
	fmt.Printf("randomly selected stream: %s\n", *srcStream.LogStreamName)

	// logGroupName/year/month/day/hour/minutes - almost the same format as CloudWatch Data Protection uses
	dstStreamName := fmt.Sprintf("%s/%s", srcGroup, time.Now().UTC().Format("2006/01/02/15/04"))
	fmt.Println("destination stream name:", dstStreamName)

	err := copyLogStream(ctx, client, srcGroup, *srcStream.LogStreamName, dstGroup, dstStreamName)
	if err != nil {
		return err
	}

	return nil
}

func copyLogStream(ctx context.Context, client *cloudwatchlogs.Client, srcGroup, srcStream, dstGroup, dstStream string) error {
	_, err := client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(dstGroup),
		LogStreamName: aws.String(dstStream),
	})
	if err != nil {
		return fmt.Errorf("create stream: %w", err)
	}

	var nextToken *string
	for {
		output, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(srcGroup),
			LogStreamName: aws.String(srcStream),
			StartFromHead: aws.Bool(true),
			NextToken:     nextToken,
		})
		if err != nil {
			return fmt.Errorf("get events: %w", err)
		}
		if len(output.Events) == 0 {
			break
		}

		inputEvents := make([]types.InputLogEvent, len(output.Events))
		for i, e := range output.Events {
			inputEvents[i] = types.InputLogEvent{
				Message:   e.Message,
				Timestamp: e.Timestamp,
			}
		}

		_, err = client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
			LogGroupName:  aws.String(dstGroup),
			LogStreamName: aws.String(dstStream),
			LogEvents:     inputEvents,
		})
		if err != nil {
			return fmt.Errorf("put events: %w", err)
		}

		// GetLogEvents returns the same token when there are no more events
		if aws.ToString(output.NextForwardToken) == aws.ToString(nextToken) {
			break
		}
		nextToken = output.NextForwardToken
	}

	return nil
}
