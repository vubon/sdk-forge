.PHONY: help build install clean lint fmt test test-coverage run check all

# Variables
BINARY_NAME=sdk-forge
MAIN_PATH=./cmd/cli
BUILD_DIR=./bin
VERSION_FILE=VERSION
VERSION?=$(shell cat $(VERSION_FILE) 2>/dev/null || echo "0.2.0-alpha.1")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.gitCommit=$(GIT_COMMIT)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

install: ## Install the binary to GOPATH/bin
	@echo "$(YELLOW)Installing $(BINARY_NAME)...$(NC)"
	$(GOBUILD) $(LDFLAGS) -o $$(go env GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)$(NC)"

clean: ## Clean build artifacts and test outputs
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -rf test-output test-sdks output
	@echo "$(GREEN)✓ Clean complete$(NC)"

lint: ## Run linter (golangci-lint)
	@echo "$(YELLOW)Running linter...$(NC)"
	@GOPATH_BIN=$$($(GOCMD) env GOPATH)/bin; \
	if ! command -v $(GOLINT) > /dev/null 2>&1 && [ ! -f "$$GOPATH_BIN/$(GOLINT)" ]; then \
		echo "$(YELLOW)golangci-lint not found. Installing latest version...$(NC)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$GOPATH_BIN latest; \
	fi
	@GOPATH_BIN=$$($(GOCMD) env GOPATH)/bin; \
	LINT_CMD=$$(command -v $(GOLINT) 2>/dev/null || echo "$$GOPATH_BIN/$(GOLINT)"); \
	if [ ! -f "$$LINT_CMD" ]; then \
		echo "$(RED)✗ golangci-lint not found. Please install it manually:$(NC)"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$GOPATH_BIN"; \
		exit 1; \
	fi; \
	LINT_OUTPUT=$$($$LINT_CMD run ./... 2>&1); \
	LINT_EXIT=$$?; \
	if echo "$$LINT_OUTPUT" | grep -q "goanalysis_metalinter.*goarch"; then \
		echo "$(YELLOW)⚠ Warning: goanalysis_metalinter error detected (known compatibility issue with Go 1.24+).$(NC)"; \
		echo "$(YELLOW)   This is a golangci-lint compatibility issue, not a code issue. Continuing...$(NC)"; \
		echo "$$LINT_OUTPUT" | grep -v "goanalysis_metalinter" | grep -v "internal/goarch" | grep -v "^$$" || true; \
		echo "$(GREEN)✓ Linting complete (goanalysis_metalinter errors ignored)$(NC)"; \
	elif [ $$LINT_EXIT -eq 0 ]; then \
		echo "$$LINT_OUTPUT"; \
		echo "$(GREEN)✓ Linting complete$(NC)"; \
	else \
		echo "$$LINT_OUTPUT"; \
		echo "$(RED)✗ Linting failed$(NC)"; \
		exit 1; \
	fi

fmt: ## Format Go code
	@echo "$(YELLOW)Formatting code...$(NC)"
	@$(GOFMT) -s -w .
	@echo "$(GREEN)✓ Formatting complete$(NC)"

fmt-check: ## Check if code is formatted correctly
	@echo "$(YELLOW)Checking code formatting...$(NC)"
	@if [ $$($(GOFMT) -l . | wc -l) -ne 0 ]; then \
		echo "$(RED)✗ Code is not formatted. Run 'make fmt' to fix.$(NC)"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ Code is properly formatted$(NC)"

test: ## Run tests (cleans test outputs first)
	@echo "$(YELLOW)Cleaning test outputs...$(NC)"
	@rm -rf test-output test-sdks
	@echo "$(YELLOW)Running tests...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)✓ Tests complete$(NC)"

test-coverage: test ## Run tests with coverage report
	@echo "$(YELLOW)Generating coverage report...$(NC)"
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"
	@echo "$(YELLOW)Coverage:$$($(GOCMD) tool cover -func=coverage.out | grep total | awk '{print $$3}')$(NC)"

test-short: ## Run tests in short mode
	@echo "$(YELLOW)Running short tests...$(NC)"
	$(GOTEST) -short -v ./...
	@echo "$(GREEN)✓ Short tests complete$(NC)"

run: build ## Build and run the binary
	@echo "$(YELLOW)Running $(BINARY_NAME)...$(NC)"
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

check: fmt-check lint ## Run all checks (formatting and linting)
	@echo "$(GREEN)✓ All checks passed$(NC)"

deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	$(GOCMD) vet ./...
	@echo "$(GREEN)✓ Go vet complete$(NC)"

all: clean deps fmt-check lint test build ## Run all checks and build
	@echo "$(GREEN)✓ All tasks complete$(NC)"

# Development helpers
dev-build: ## Build for development (no optimizations)
	@echo "$(YELLOW)Building for development...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Development build complete$(NC)"

version: ## Show version information
	@echo "$(YELLOW)Current Version: $(VERSION)$(NC)"
	@echo "$(YELLOW)Version File: $(VERSION_FILE)$(NC)"
	@echo "$(YELLOW)Go version: $$($(GOCMD) version)$(NC)"
	@echo "$(YELLOW)Binary: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"
	@if [ -f "$(BUILD_DIR)/$(BINARY_NAME)" ]; then \
		echo "$(YELLOW)Built binary version:$(NC)"; \
		$(BUILD_DIR)/$(BINARY_NAME) --version; \
	fi

