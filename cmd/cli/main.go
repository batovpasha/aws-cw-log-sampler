package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cli"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
	"golang.org/x/sync/errgroup"
)

// GetLogEvents has the lowest TPS - 10, and it's a bottleneck.
// See limits of other APIs here: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const concurrencyLimit = 10

func main() {
	fs := flag.CommandLine
	flags := cli.RegisterCommonFlags(fs)
	lookbackHours := fs.Int("lookbackHours", 24, "lookback hours")
	fs.Parse(os.Args[1:])

	err := cli.ValidateCommonFlags(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *lookbackHours <= 0 {
		fmt.Fprintln(os.Stderr, "--lookbackHours should be an integer")
		os.Exit(1)
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	srcGroups, err := cloudwatchlogs.ListLogGroupNames(ctx, client, flags.LogGroupNamePattern)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var g errgroup.Group
	g.SetLimit(concurrencyLimit)

	cutoff := time.Now().Add(-time.Duration(*lookbackHours) * time.Hour).UnixMilli()

	for _, srcGroup := range srcGroups {
		g.Go(func() error {
			if err := sample.ProcessLogGroup(ctx, client, cutoff, srcGroup, flags.DstGroup); err != nil {
				fmt.Printf("error processing log group %s: %v\n", srcGroup, err)
			}
			return nil
		})
	}

	_ = g.Wait()
}
