package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/batovpasha/aws-cw-log-sampler/internal/sample"
	"golang.org/x/sync/errgroup"
)

// GetLogEvents has the lowest TPS - 10, and it's a bottleneck.
// See limits of other APIs here: https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html
const concurrencyLimit = 10

type Config struct {
	SrcGroups []string `json:"srcGroups"`
	DstGroup  string   `json:"dstGroup"`
}

func main() {
	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	var appCfg Config
	if err := json.Unmarshal(data, &appCfg); err != nil {
		log.Fatal(err)
	}
	fmt.Println(appCfg)

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	client := cloudwatchlogs.NewFromConfig(cfg)

	var g errgroup.Group
	g.SetLimit(concurrencyLimit)

	for _, srcGroup := range appCfg.SrcGroups {
		g.Go(func() error {
			if err := sample.ProcessLogGroup(ctx, client, srcGroup, appCfg.DstGroup); err != nil {
				fmt.Printf("error processing log group %s: %v\n", srcGroup, err)
			}
			return nil
		})
	}

	_ = g.Wait()
}
