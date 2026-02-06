BINARY    := rp
BUILD_DIR := .
CMD_DIR   := ./cmd/$(BINARY)

.DEFAULT_GOAL := help

.PHONY: all
all: tidy generate lint build ## Run tidy, generate, lint, and build

.PHONY: build
build: ## Build the rp binary
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

.PHONY: vet
vet: ## Run go vet on all packages
	go vet ./...

.PHONY: generate
generate: ## Run go generate for all packages
	go generate ./...

.PHONY: test
test: ## Run unit tests
	go test ./... ./pkg/... -count=1

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	cd test/integration && go test -tags=integration -v -count=1 -timeout 10m ./...

.PHONY: test-all
test-all: test test-integration ## Run all tests

.PHONY: tidy
tidy: ## Run go mod tidy for all modules
	go mod tidy
	cd pkg && go mod tidy
	cd test/integration && go mod tidy

.PHONY: lint
lint: ## Run golangci-lint and auto-fix issues
	golangci-lint-v2 run --fix

.PHONY: clean
clean: ## Remove the built binary
	rm -f $(BUILD_DIR)/$(BINARY)

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
