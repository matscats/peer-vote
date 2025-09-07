# Peer-Vote Makefile

# Variables
APP_NAME := peer-vote
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | cut -d ' ' -f 3)

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)"

# Directories
BUILD_DIR := build
DIST_DIR := dist
DOCS_DIR := docs

# Go related variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo "Peer-Vote - Decentralized Voting System"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the application
.PHONY: build
build: clean
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

## build-all: Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./cmd/$(APP_NAME)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 ./cmd/$(APP_NAME)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/$(APP_NAME)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/$(APP_NAME)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/$(APP_NAME)
	
	@echo "Cross-compilation complete. Binaries in $(DIST_DIR)/"

## run: Run the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

## run-dev: Run in development mode
.PHONY: run-dev
run-dev:
	@echo "Running $(APP_NAME) in development mode..."
	$(GOCMD) run ./cmd/$(APP_NAME) -log-level debug

## test: Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-unit: Run unit tests only
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -short ./...

## test-integration: Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -run Integration ./test/integration/...

## test-e2e: Run end-to-end tests
.PHONY: test-e2e
test-e2e:
	@echo "Running end-to-end tests..."
	$(GOTEST) -v -race ./test/e2e/...

## coverage: Generate test coverage report
.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## benchmark: Run benchmarks
.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## lint: Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) mod tidy

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## check: Run all checks (fmt, vet, lint, test)
.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

## deps: Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## deps-update: Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html

## install: Install the application
.PHONY: install
install: build
	@echo "Installing $(APP_NAME)..."
	cp $(BUILD_DIR)/$(APP_NAME) $(GOPATH)/bin/

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

## docker-run: Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 -p 4001:4001 $(APP_NAME):latest

## docs: Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@mkdir -p $(DOCS_DIR)
	$(GOCMD) doc -all ./... > $(DOCS_DIR)/api.txt
	@echo "Documentation generated in $(DOCS_DIR)/"

## setup-dev: Setup development environment
.PHONY: setup-dev
setup-dev:
	@echo "Setting up development environment..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@mkdir -p data/blockchain data/storage keys
	@echo "Development environment setup complete!"

## version: Show version information
.PHONY: version
version:
	@echo "$(APP_NAME) version $(VERSION)"
	@echo "Build time: $(BUILD_TIME)"
	@echo "Go version: $(GO_VERSION)"

## init-node: Initialize a new node
.PHONY: init-node
init-node: build
	@echo "Initializing new node..."
	./$(BUILD_DIR)/$(APP_NAME) init

## start-validator: Start node as validator
.PHONY: start-validator
start-validator: build
	@echo "Starting validator node..."
	./$(BUILD_DIR)/$(APP_NAME) -config configs/validator.yaml

## start-peer: Start node as peer
.PHONY: start-peer
start-peer: build
	@echo "Starting peer node..."
	./$(BUILD_DIR)/$(APP_NAME) -config configs/peer.yaml

## create-election: Create a test election
.PHONY: create-election
create-election:
	@echo "Creating test election..."
	curl -X POST http://localhost:8080/api/v1/elections \
		-H "Content-Type: application/json" \
		-d '{"title":"Test Election","description":"A test election","candidates":[{"id":"1","name":"Candidate 1"},{"id":"2","name":"Candidate 2"}],"start_time":"2024-01-01T00:00:00Z","end_time":"2024-12-31T23:59:59Z"}'

## vote: Submit a test vote
.PHONY: vote
vote:
	@echo "Submitting test vote..."
	curl -X POST http://localhost:8080/api/v1/votes \
		-H "Content-Type: application/json" \
		-d '{"election_id":"election_hash_here","candidate_id":"1"}'

## status: Check node status
.PHONY: status
status:
	@echo "Checking node status..."
	curl -s http://localhost:8080/health | jq .

## example-blockchain: Run blockchain example (Phase 2)
.PHONY: example-blockchain
example-blockchain:
	@echo "Running blockchain example..."
	$(GOCMD) run ./examples/blockchain_example.go

## example-consensus: Run consensus PoA example (Phase 3)
.PHONY: example-consensus
example-consensus:
	@echo "Running consensus PoA example..."
	$(GOCMD) run ./examples/consensus_example.go

## example-p2p: Run P2P network example (Phase 4)
.PHONY: example-p2p
example-p2p:
	@echo "Running P2P network example..."
	$(GOCMD) run ./examples/p2p_example.go

## example-voting: Run voting system example
.PHONY: example-voting
example-voting:
	@echo "Running voting system example..."
	$(GOCMD) run ./examples/voting_example.go

## example-complete: Run complete voting simulation
.PHONY: example-complete
example-complete:
	@echo "Running complete voting simulation..."
	$(GOCMD) run ./examples/complete_voting_simulation.go

# CLI Commands
## cli-help: Show CLI help
.PHONY: cli-help
cli-help: build
	@echo "Showing CLI help..."
	./$(BUILD_DIR)/$(APP_NAME) --help

## cli-start: Start peer-vote node
.PHONY: cli-start
cli-start: build
	@echo "Starting peer-vote node..."
	./$(BUILD_DIR)/$(APP_NAME) start

## cli-status: Show system status
.PHONY: cli-status
cli-status: build
	@echo "Checking system status..."
	./$(BUILD_DIR)/$(APP_NAME) status

## cli-version: Show version
.PHONY: cli-version
cli-version: build
	@echo "Showing version..."
	./$(BUILD_DIR)/$(APP_NAME) version

## example-all: Run all examples
.PHONY: example-all
example-all: example-blockchain example-consensus example-p2p example-voting example-complete

# Development shortcuts
dev: run-dev
test-all: test
build-release: build-all
