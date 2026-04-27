VERSION := $(shell git describe --tags --long --dirty --always)

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build-cli
build-cli:
	go build -ldflags "-X main.version=$(VERSION)" -o cli ./cmd/cli

.PHONY: docker-cli
docker-cli:
	docker build -t aws-cw-log-sampler-cli:latest . -f build/Dockerfile --build-arg VERSION=$(VERSION)