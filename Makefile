VERSION := $(shell git describe --tags --long --dirty --always)

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build-cli
build-cli:
	go build -ldflags "-X main.version=$(VERSION)" -o cli ./cmd/cli

.PHONY: docker-cli
docker-cli:
	docker build -t aws-cw-log-sampler-cli:latest . -f build/cli.Dockerfile --build-arg VERSION=$(VERSION)

.PHONY: docker-service
docker-service:
	docker build -t aws-cw-log-sampler:latest . -f build/service.Dockerfile --build-arg VERSION=$(VERSION)
