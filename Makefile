GO_BIN       ?= go
CLI_NAME     := mump2p
BUILD_DIR    := dist

VERSION      ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT_HASH  ?= $(shell git rev-parse --short HEAD)
DOMAIN       ?= ""
CLIENT_ID    ?= ""
AUDIENCE     ?= optimum-login
SERVICE_URL  ?= http://localhost:12080

LD_FLAGS := -X github.com/getoptimum/mump2p-cli/internal/config.Domain=$(DOMAIN) \
            -X github.com/getoptimum/mump2p-cli/internal/config.ClientID=$(CLIENT_ID) \
            -X github.com/getoptimum/mump2p-cli/internal/config.Audience=$(AUDIENCE) \
            -X github.com/getoptimum/mump2p-cli/internal/config.ServiceURL=$(SERVICE_URL) \
            -X github.com/getoptimum/mump2p-cli/internal/config.Version=$(VERSION) \
            -X github.com/getoptimum/mump2p-cli/internal/config.CommitHash=$(COMMIT_HASH)

.PHONY: all build run clean test help lint build tag release print-cli-name e2e-test e2e-quick

all: lint build

lint: ## Run linter
	golangci-lint run ./...

build: ## Build the CLI binary
	GOOS=darwin GOARCH=amd64 $(GO_BIN) build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(CLI_NAME)-mac .
	GOOS=linux GOARCH=amd64 $(GO_BIN) build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(CLI_NAME)-linux .

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
	
run: build ## Run the CLI with default config
	./$(CLI_NAME) --config=$(CONFIG_PATH)

run-subscribe: build ## Run subscribe command
	./$(CLI_NAME) subscribe --topic=demo --protocols=optimump2p --config=$(CONFIG_PATH)

run-publish: build ## Run publish command
	./$(CLI_NAME) publish --topic=demo --protocols=optimump2p --config=$(CONFIG_PATH)

clean: ## Clean up build artifacts
	rm -f $(CLI_NAME)

test: ## Run unit tests
	$(GO_BIN) test ./... -v -count=1

e2e-test: ## Run E2E tests against dist/ binary
	@echo "Running E2E tests..."
	@if [ ! -f "$(BUILD_DIR)/$(CLI_NAME)-linux" ] && [ ! -f "$(BUILD_DIR)/$(CLI_NAME)-mac" ]; then \
		echo "Error: No binary found in $(BUILD_DIR)/"; \
		echo "Run 'make build' first with release credentials"; \
		exit 1; \
	fi
	go test ./e2e -v -timeout 10m

e2e-quick: ## Run quick smoke tests only
	@echo "Running quick smoke tests..."
	go test ./e2e -v -run TestCLISmokeCommands -timeout 2m

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'