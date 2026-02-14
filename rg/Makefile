# Makefile for rg & sd Mastery project
# A Go webshop codebase used as a practice target for ripgrep and sd exercises.

BINARY_NAME := webshop
BUILD_DIR := bin
GO := go
RG := rg
SD := sd

.PHONY: build test run lint clean seed-db migrate fmt vet todo fixme hack deprecated audit secrets help

## --- Core Targets ---

build: ## Build the webshop binary
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/webshop/

test: ## Run all tests
	$(GO) test -v -race ./...

run: build ## Build and run the webshop server
	./$(BUILD_DIR)/$(BINARY_NAME)

lint: vet ## Run linters (go vet + staticcheck if available)
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Format all Go source files
	$(GO) fmt ./...

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	$(GO) clean -cache

## --- Database Targets ---

migrate: ## Run database migrations
	@echo "Running migrations..."
	psql -U devuser -d webshop_dev -f scripts/migrate.sql

seed-db: ## Seed the database with test data
	@echo "Seeding database..."
	psql -U devuser -d webshop_dev -f scripts/seed.sql

## --- Codebase Search Targets (rg + sd practice) ---

todo: ## List all TODO comments in the codebase
	@$(RG) 'TODO:' --color always -n || echo "No TODOs found"

fixme: ## List all FIXME comments in the codebase
	@$(RG) 'FIXME:' --color always -n || echo "No FIXMEs found"

hack: ## List all HACK comments in the codebase
	@$(RG) 'HACK:' --color always -n || echo "No HACKs found"

deprecated: ## Find all DEPRECATED markers
	@$(RG) 'DEPRECATED' --color always -n || echo "No DEPRECATED markers found"

audit: ## Audit codebase for common issues
	@echo "=== Hardcoded Secrets ==="
	@$(RG) -i '(password|secret|api.?key)\s*[:=]\s*"[^$]' --type go --type yaml -n || echo "  None found"
	@echo ""
	@echo "=== TODO/FIXME/HACK Summary ==="
	@echo "  TODOs:  $$($(RG) -c 'TODO' --type go --type yaml --type sql | awk -F: '{s+=$$2} END {print s+0}')"
	@echo "  FIXMEs: $$($(RG) -c 'FIXME' --type go --type yaml --type sql | awk -F: '{s+=$$2} END {print s+0}')"
	@echo "  HACKs:  $$($(RG) -c 'HACK' --type go --type yaml --type sql | awk -F: '{s+=$$2} END {print s+0}')"

secrets: ## Scan for potential secrets and credentials
	@echo "Scanning for hardcoded credentials..."
	@$(RG) -i '(password|passwd|secret|token|api_key|apikey)\s*[:=]' -n --color always || echo "No potential secrets found"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
