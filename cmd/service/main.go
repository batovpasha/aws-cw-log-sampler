package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/robfig/cron/v3"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cli"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cloudwatchlogs"
	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
)

func main() {
	fs := flag.CommandLine
	flags := cli.RegisterCommonFlags(fs)
	cronExpr := fs.String("cron", "", "cron expression")
	_ = fs.Parse(os.Args[1:])

	err := cli.ValidateCommonFlags(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *cronExpr == "" {
		fmt.Fprintln(os.Stderr, "--cron is required")
		os.Exit(1)
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(*cronExpr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	now := time.Now()
	next1 := sched.Next(now)
	next2 := sched.Next(next1)
	interval := next2.Sub(next1)
	cutoff := now.Add(-interval).UnixMilli()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	err = sample.Sample(ctx, client, &sample.Config{
		LogGroupNamePattern: flags.LogGroupNamePattern,
		DstGroup:            flags.DstGroup,
		Type:                flags.Type,
		Cutoff:              cutoff,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
