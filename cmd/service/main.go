package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	ctx, cancel := context.WithCancel(context.Background())
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	now := time.Now()
	next1 := sched.Next(now)
	next2 := sched.Next(next1)
	interval := next2.Sub(next1)

	errCh := make(chan error, 1)

	c := cron.New()
	_, err = c.AddFunc(*cronExpr, func() {
		cutoff := time.Now().Add(-interval).UnixMilli()
		err := sample.Sample(ctx, client, &sample.Config{
			LogGroupNamePattern:  flags.LogGroupNamePattern,
			DstGroup:             flags.DstGroup,
			Type:                 flags.Type,
			Cutoff:               cutoff,
			RandLogStreamsNumber: flags.RandLogStreamsNumber,
		})
		if err != nil {
			errCh <- err
			return
		}
		log.Printf("The next run is scheduled on: %v", c.Entries()[0].Next)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c.Start()
	log.Printf("The next run is scheduled on: %v", c.Entries()[0].Next)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	exitCode := 0
	select {
	case <-sigCh:
		// normal shutdown
	case err = <-errCh:
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
	}

	cancel()
	c.Stop()
	os.Exit(exitCode)
}
