package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cli"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cwlogs"
	"github.com/batovpasha/aws-cw-log-sampler/internal/logger"
	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
)

// Version is set at build time using -ldflags "-X main.version=$VERSION"
var version = "dev"

func main() {
	fs := flag.CommandLine
	flags := cli.RegisterCommonFlags(fs)
	lookbackHours := fs.Int("lookbackHours", 24, "lookback hours")
	_ = fs.Parse(os.Args[1:])

	err := cli.ValidateCommonFlags(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *lookbackHours <= 0 {
		fmt.Fprintln(os.Stderr, "--lookbackHours should be an integer")
		os.Exit(1)
	}

	logger.New(version)
	ctx := logger.WithTraceID(context.Background())
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := cwlogs.NewFromConfig(cfg)

	cutoff := time.Now().Add(-time.Duration(*lookbackHours) * time.Hour).UnixMilli()

	err = sample.Sample(ctx, client, &sample.Config{
		LogGroupNamePattern:  flags.LogGroupNamePattern,
		DstGroup:             flags.DstGroup,
		Type:                 flags.Type,
		Cutoff:               cutoff,
		RandLogStreamsNumber: flags.RandLogStreamsNumber,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
