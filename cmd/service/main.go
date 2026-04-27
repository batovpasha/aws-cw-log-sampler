package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/robfig/cron/v3"

	"github.com/batovpasha/aws-cw-log-sampler/internal/cli"
	"github.com/batovpasha/aws-cw-log-sampler/internal/cwlogs"
	"github.com/batovpasha/aws-cw-log-sampler/internal/logger"
	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
)

// Version is set at build time using -ldflags "-X main.version=$VERSION"
var version string // do not remove or modify

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

	logger.New(version)
	ctx, cancel := context.WithCancel(context.Background())
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := cwlogs.NewFromConfig(cfg)

	now := time.Now()
	next1 := sched.Next(now)
	next2 := sched.Next(next1)
	interval := next2.Sub(next1)

	c := cron.New()
	_, err = c.AddFunc(*cronExpr, func() {
		runCtx := logger.WithTraceID(ctx)
		cutoff := time.Now().Add(-interval).UnixMilli()
		err := sample.Sample(runCtx, client, &sample.Config{
			LogGroupNamePattern:  flags.LogGroupNamePattern,
			DstGroup:             flags.DstGroup,
			Type:                 flags.Type,
			Cutoff:               cutoff,
			RandLogStreamsNumber: flags.RandLogStreamsNumber,
		})
		if err != nil {
			slog.WarnContext(runCtx, "error during sampling", "error", err)
		}
		slog.Info("schedule next run", "time", c.Entries()[0].Next)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c.Start()
	slog.Info("schedule next run", "time", c.Entries()[0].Next)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	cancel()
	c.Stop()
}
