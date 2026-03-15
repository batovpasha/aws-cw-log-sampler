# aws-cw-log-sampler

Golang CLI/service for CloudWatch log sampling

## cmd/cli

Use cases:

- Schedule sampling by external scheduler (e.g., Linux cron/systemd timers, Kubernetes CronJob,
  Cloud schedulers like EventBridge)
- Run on-demand
- Troubleshooting/Debugging

## cmd/service

Use cases:

- Launch as a long-running service(e.g., on a VPS or locally), when you already have a machine that
  runs continuously and you don't care about the per-run costs of a serverless solution
