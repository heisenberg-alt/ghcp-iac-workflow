.PHONY: build test lint run dev docker docker-run clean fmt vet

BINARY_NAME=ghcp-iac-server
CLI_NAME=gh-iac
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Build
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/server
	go build $(LDFLAGS) -o bin/$(CLI_NAME) ./cmd/gh-iac

build-server:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/server

build-cli:
	go build $(LDFLAGS) -o bin/$(CLI_NAME) ./cmd/gh-iac

# Test
test:
	go test -v -race -count=1 ./internal/...

test-integration:
	go test -v -race -count=1 -tags=integration ./cmd/server/...

test-cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./internal/... ./cmd/...
	go tool cover -func=coverage.out
	@echo "---"
	@go tool cover -func=coverage.out | grep total

test-cover-html: test-cover
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

# Lint
lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

# Run
run: build-server
	./bin/$(BINARY_NAME)

dev:
	go run ./cmd/server

# Docker
docker:
	docker build -t ghcp-iac:$(VERSION) -t ghcp-iac:latest .

docker-run:
	docker run -p 8080:8080 --env-file .env ghcp-iac:latest

# Deploy
terraform-plan-%:
	cd deploy/terraform/environments/$* && terraform plan

terraform-apply-%:
	cd deploy/terraform/environments/$* && terraform apply

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html
