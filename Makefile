GO_BIN ?= go
CLI_NAME := mump2p
CONFIG_PATH ?= app_conf.yml

.PHONY: all build run clean test help

all: build

build: ## Build the CLI binary
	$(GO_BIN) build -o $(CLI_NAME) .

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

help: ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
