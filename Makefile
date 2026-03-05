APP_NAME ?= rtt3168ctl
CMD_DIR ?= ./cmd/rtt3168ctl
BIN_DIR ?= ./build
BIN ?= $(BIN_DIR)/$(APP_NAME)

GO ?= go
GOFLAGS ?=
LDFLAGS ?=
ARGS ?=

all: build

.PHONY: help build run test fmt vet tidy clean install

help: ## Show available targets 
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the CLI binary (Default)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD_DIR)

run: ## Run app (use ARGS="...")
	$(GO) run $(CMD_DIR) $(ARGS)

test: ## Run tests
	$(GO) test ./...

fmt: ## Format Go code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

tidy: ## Tidy Go modules
	$(GO) mod tidy

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) ./$(APP_NAME)

install: ## Install binary to GOPATH/bin
	$(GO) install $(CMD_DIR)
