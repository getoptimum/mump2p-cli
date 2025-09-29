-include .env # inclusion is optional
export
COVERAGE_THRESHOLD := 20
COVERPROFILE       := coverage.out
GO_BIN             := go
CLI_NAME           := mump2p
BUILD_DIR          := dist

VERSION      := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT_HASH  := $(shell git rev-parse --short HEAD)
DOMAIN       ?= "localhost"
CLIENT_ID    ?= "default-client"
AUDIENCE     ?= optimum-login
SERVICE_URL  ?= http://localhost:12080

UNAME_S := $(shell uname)
ifeq ($(UNAME_S),Darwin)
	HOST_OS := mac
else
	HOST_OS := linux
endif
CLI_BINARY := $(BUILD_DIR)/$(CLI_NAME)-$(HOST_OS)

GOLANGCI_VER := v2.1.1
VULNCHECK_VER := v1.1.4
VULNCHECK_RUN := $(GO_BIN) run golang.org/x/vuln/cmd/govulncheck@$(VULNCHECK_VER)

LD_FLAGS := -X github.com/getoptimum/mump2p-cli/internal/config.Domain=$(DOMAIN) \
            -X github.com/getoptimum/mump2p-cli/internal/config.ClientID=$(CLIENT_ID) \
            -X github.com/getoptimum/mump2p-cli/internal/config.Audience=$(AUDIENCE) \
            -X github.com/getoptimum/mump2p-cli/internal/config.ServiceURL=$(SERVICE_URL) \
            -X github.com/getoptimum/mump2p-cli/internal/config.Version=$(VERSION) \
            -X github.com/getoptimum/mump2p-cli/internal/config.CommitHash=$(COMMIT_HASH)

.PHONY: all build run clean test help lint tag release print-cli-name vulcheck tools ci coverage debug-vars

all: lint build
ci: lint test coverage vulcheck
lint: ## Run linter
	golangci-lint run ./...

tools: ## Install dev/CI tools (govulncheck, golangci-lint)
	@echo "🔧 installing tools"
	@go install golang.org/x/vuln/cmd/govulncheck@$(VULNCHECK_VER)
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VER)


define build_template
GOOS=$(1) GOARCH=amd64 $(GO_BIN) build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(CLI_NAME)-$(2) .
endef

build: build-linux build-mac ## Build both OS binaries

build-linux: ## Build linux binary
	@echo "Building Linux binary..."
	$(call build_template,linux,linux)

build-mac: ## Build macOS binary
	@echo "Building macOS binary..."
	$(call build_template,darwin,mac)

build-local: ## Build for the local OS
	@echo "Building for local OS: $(HOST_OS)"
	$(MAKE) --no-print-directory build-$(HOST_OS)

print-cli-name: ## Print CLI name for CI/CD usage
	@echo -n "$(CLI_NAME)"

release: build ## Build and create GitHub release
	@echo "Creating release for $(VERSION)"
	mkdir -p $(BUILD_DIR)
	gh release create $(VERSION) \
		--title "Release $(VERSION)" \
		--notes "Release notes for $(VERSION)" \
		$(BUILD_DIR)/$(CLI_NAME)-mac \
		$(BUILD_DIR)/$(CLI_NAME)-linux \
		
tag:
	@echo "Calculating next RC tag..."
	@latest_tag=$$(git tag --sort=-creatordate | grep '^v0\.' | grep -E 'rc[0-9]+$$' | head -n1); \
	if [ -z "$$latest_tag" ]; then \
		new_tag="v0.0.1-rc4"; \
	else \
		version=$$(echo $$latest_tag | sed -E 's/^v([0-9]+\.[0-9]+\.[0-9]+)-rc([0-9]+)$$/\1/'); \
		rc_num=$$(echo $$latest_tag | sed -E 's/^v[0-9]+\.[0-9]+\.[0-9]+-rc([0-9]+)$$/\1/'); \
		new_rc_num=$$(expr $$rc_num + 1); \
		new_tag="v$$version-rc$$new_rc_num"; \
	fi; \
	echo "New tag: $$new_tag"; \
	git tag -a $$new_tag -m "Release $$new_tag"; \
	git push origin $$new_tag
	
subscribe: build ## Run subscribe command against default local  proxy
	./$(CLI_NAME) subscribe --topic=demo

publish: build ## Run publish command against default local proxy
	./$(CLI_NAME) publish --topic=demo


subscribe_against_remote_proxy: ## loads REMOTE_PROXY IP from .env for SERVICE_URL
	./$(CLI_BINARY) subscribe --topic=walentyn --service-url=$(REMOTE_PROXY_URL_1)

publish_against_remote_proxy: ## loads REMOTE_PROXY IP from .env for SERVICE_URL
	./$(CLI_BINARY) publish --topic=walentyn --message="cool to be here" --service-url=$(REMOTE_PROXY_URL_2)

clean:
	@echo "Cleaning build artifacts in $(BUILD_DIR)..."
	rm -rf "$(BUILD_DIR)"/*

test: ## Run unit tests
	$(GO_BIN) test ./... -v -count=1 -covermode=atomic -coverprofile=coverage.out

coverage: $(COVERPROFILE) ## Enforce threshold
	@echo "📈 coverage:"
	@$(GO_BIN) tool cover -func=coverage.out | tail -1
	@total=$$($(GO_BIN) tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%'); \
	 echo "Threshold: $(COVERAGE_THRESHOLD)%"; echo "Actual: $$total%"; \
	 awk 'BEGIN{exit !(('$$total') >= $(COVERAGE_THRESHOLD))}' || (echo "❌ Coverage too low!" && exit 1)

coverhtml: $(COVERPROFILE) ## Generate HTML coverage report
	@$(GO_BIN) tool cover -html=$(COVERPROFILE) -o coverage.html
	@echo "📊 open coverage.html to view the report"

vulcheck: ## Run govulncheck
	@echo "🔍 Running govulncheck..."
	@$(VULNCHECK_RUN) ./...

debug-vars:
	@echo "DOMAIN             = $(DOMAIN)"
	@echo "CLIENT_ID          = $(CLIENT_ID)"
	@echo "AUDIENCE           = $(AUDIENCE)"
	@echo "SERVICE_URL        = $(SERVICE_URL)"
	@echo "REMOTE_PROXY_URL_1 = $(REMOTE_PROXY_URL_1)"
	@echo "REMOTE_PROXY_URL_2 = $(REMOTE_PROXY_URL_2)"

help: ## Show help
	@echo ""
	@echo "📘 Available targets:"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' Makefile | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@echo ""

# guard target
$(COVERPROFILE):
	@echo "No $(COVERPROFILE) found; run 'make test' first." >&2; exit 1

.DEFAULT_GOAL := help

e2e-test:
	go run ./e2e

set-auth0-token-ci:
	gh secret set AUTH0_TOKEN < ~/.mump2p/auth.yml
