# Makefile for OpenSearch Query Receiver

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=otelcol-opensearchquery
BINARY_PATH=./bin/$(BINARY_NAME)

# Directories
CMD_DIR=./cmd/otelcol
DIST_DIR=./dist

# Build variables
VERSION?=0.1.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

.PHONY: all
all: fmt vet test build

.PHONY: help
help: ## Display this help message
	@echo "OpenSearch Query Receiver - Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	@$(GOFMT) ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@$(GOVET) ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint installed)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run ./...

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy

.PHONY: test
test: ## Run unit tests
	@echo "Running tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...

.PHONY: test-short
test-short: ## Run short tests (skip integration tests)
	@echo "Running short tests..."
	@$(GOTEST) -v -short -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker services)
	@echo "Running integration tests..."
	@$(GOTEST) -v -tags=integration ./...

.PHONY: coverage
coverage: test ## Generate and display test coverage report
	@echo "Generating coverage report..."
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: build
build: deps ## Build the collector binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(CMD_DIR)/main.go
	@echo "Binary built: $(BINARY_PATH)"

.PHONY: build-linux
build-linux: deps ## Build for Linux amd64
	@echo "Building for Linux amd64..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-linux-amd64 $(CMD_DIR)/main.go
	@echo "Binary built: $(BINARY_PATH)-linux-amd64"

.PHONY: build-darwin
build-darwin: deps ## Build for macOS (Darwin) amd64 and arm64
	@echo "Building for macOS..."
	@mkdir -p bin
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-darwin-amd64 $(CMD_DIR)/main.go
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-darwin-arm64 $(CMD_DIR)/main.go
	@echo "Binaries built: $(BINARY_PATH)-darwin-{amd64,arm64}"

.PHONY: build-all
build-all: build-linux build-darwin ## Build for all platforms

.PHONY: install
install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

.PHONY: docker-up
docker-up: ## Start Docker test environment
	@echo "Starting Docker test environment..."
	@cd testdata && docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Docker environment ready!"
	@echo "  - OpenSearch: https://localhost:9200 (admin/admin)"
	@echo "  - OpenSearch Dashboards: http://localhost:5601"
	@echo "  - OAuth Proxy: http://localhost:8080"

.PHONY: docker-down
docker-down: ## Stop Docker test environment
	@echo "Stopping Docker test environment..."
	@cd testdata && docker-compose down -v
	@echo "Docker environment stopped"

.PHONY: docker-logs
docker-logs: ## Show Docker logs
	@cd testdata && docker-compose logs -f

.PHONY: docker-restart
docker-restart: docker-down docker-up ## Restart Docker test environment

.PHONY: run-direct
run-direct: build ## Run collector with direct mode configuration
	@echo "Running collector in direct mode..."
	@$(BINARY_PATH) --config=examples/config-direct.yaml

.PHONY: run-proxy
run-proxy: build ## Run collector with proxy mode configuration
	@echo "Running collector in proxy mode..."
	@$(BINARY_PATH) --config=examples/config-proxy.yaml

.PHONY: test-query
test-query: ## Test OpenSearch query (requires running OpenSearch)
	@echo "Testing OpenSearch query..."
	@curl -k -u admin:admin -X GET "https://localhost:9200/_cluster/health?pretty"

.PHONY: test-proxy
test-proxy: ## Test OAuth proxy (requires running proxy)
	@echo "Testing OAuth proxy..."
	@curl -X GET "http://localhost:8080/_cluster/health?pretty"

.PHONY: builder
builder: ## Build using OpenTelemetry Collector Builder
	@echo "Building with OpenTelemetry Collector Builder..."
	@which ocb > /dev/null || (echo "ocb not installed. Install from https://github.com/open-telemetry/opentelemetry-collector/releases" && exit 1)
	@ocb --config=builder-config.yaml
	@echo "Custom collector built in $(DIST_DIR)/"

.PHONY: generate
generate: ## Run go generate
	@echo "Running go generate..."
	@$(GOCMD) generate ./...

.PHONY: mod-update
mod-update: ## Update Go module dependencies
	@echo "Updating dependencies..."
	@$(GOMOD) get -u ./...
	@$(GOMOD) tidy

.PHONY: check
check: fmt vet test ## Run all checks (fmt, vet, test)

.PHONY: ci
ci: deps check build ## Run CI pipeline (deps, checks, build)

.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@which godoc > /dev/null || (echo "godoc not installed. Run: go install golang.org/x/tools/cmd/godoc@latest" && exit 1)
	@echo "Starting godoc server on http://localhost:6060"
	@godoc -http=:6060

.PHONY: dev
dev: docker-up build run-direct ## Start development environment

.DEFAULT_GOAL := help
