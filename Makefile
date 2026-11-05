GO ?= go
BINARY := gg-lint

.DEFAULT_GOAL := help

.PHONY: help fmt build run test clean

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*?##/ {printf "  %-10s %s", $$1, $$2; print ""}' $(MAKEFILE_LIST)

fmt: ## Format Go code and run go vet
	$(GO) fmt ./...
	$(GO) vet ./...

build: fmt ## Build the gg-lint binary
	CGO_ENABLED=0 $(GO) build -o $(BINARY) ./cmd/gg-lint

test: fmt ## Run the test suite
	$(GO) test ./...

clean: ## Remove the built binary
	rm -f $(BINARY)
