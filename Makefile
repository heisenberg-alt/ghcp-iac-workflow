.PHONY: build test lint run dev docker docker-run clean fmt vet

BINARY_NAME=ghcp-iac-server
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Build
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/agent-host

# Test
test:
	go test -v -race -count=1 ./...

test-agents:
	go test -v -race -count=1 ./agents/...

test-integration:
	go test -v -race -count=1 -tags=integration ./cmd/...

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
run: build
	./bin/$(BINARY_NAME)

dev:
	go run ./cmd/agent-host

dev-mcp:
	go run ./cmd/agent-host -- --transport=stdio

# Docker
docker:
	docker build -t ghcp-iac:$(VERSION) -t ghcp-iac:latest --build-arg BUILD_CMD=agent-host .

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
