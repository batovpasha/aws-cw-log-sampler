package cwlogs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type Client = cloudwatchlogs.Client

var NewFromConfig = cloudwatchlogs.NewFromConfig

// DescribeLogStreamsUntilCutoff lists all log streams in the specified log group that have
// a last event timestamp greater than or equal to the cutoff.
//
// cutoff is a Unix time in milliseconds.
//
// Doesn't perform pagination, returns all log streams in one slice.
func DescribeLogStreamsUntilCutoff(
	ctx context.Context,
	client *cloudwatchlogs.Client,
	logGroupName string,
	cutoff int64,
) ([]types.LogStream, error) {
	var nextToken *string
	var logStreams []types.LogStream

Outer:
	for {
		output, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: aws.String(logGroupName),
			OrderBy:      types.OrderByLastEventTime,
			Descending:   aws.Bool(true),
			NextToken:    nextToken,
			Limit:        aws.Int32(50),
		})
		if err != nil {
			return nil, fmt.Errorf("describe log streams: %w", err)
		}
		if len(output.LogStreams) == 0 {
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

	return logStreams, nil
}

// CopyLogStream copies log events from the source log stream to the destination log stream.
//
// The dstGroup must already exist, the dstStream will be created by this function.
func CopyLogStream(
	ctx context.Context,
	client *cloudwatchlogs.Client,
	srcGroup, srcStream, dstGroup, dstStream string,
) (processed int, err error) {
	_, err = client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(dstGroup),
		LogStreamName: aws.String(dstStream),
	})
	if err != nil {
		return processed, fmt.Errorf("create stream: %w", err)
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
			return processed, fmt.Errorf("get events: %w", err)
		}
		if len(output.Events) == 0 {
			break
		}
		processed += len(output.Events)

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
			return processed, fmt.Errorf("put events: %w", err)
		}

		// GetLogEvents returns the same token when there are no more events
		if aws.ToString(output.NextForwardToken) == aws.ToString(nextToken) {
			break
		}
		nextToken = output.NextForwardToken
	}

	return processed, nil
}

// ListLogGroupNames lists log group names that match the specified pattern.
//
// Doesn't perform pagination, returns all log group names in one slice.
func ListLogGroupNames(
	ctx context.Context,
	client *cloudwatchlogs.Client,
	logGroupNamePattern string,
) ([]string, error) {
	var nextToken *string
	var logGroups []string

	for {
		output, err := client.ListLogGroups(ctx, &cloudwatchlogs.ListLogGroupsInput{
			LogGroupNamePattern: aws.String(logGroupNamePattern),
			NextToken:           nextToken,
			Limit:               aws.Int32(1000),
		})
		if err != nil {
			return nil, fmt.Errorf("list log groups: %w", err)
		}

		for _, group := range output.LogGroups {
			logGroups = append(logGroups, aws.ToString(group.LogGroupName))
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}

	return logGroups, nil
}
