GO_BIN       ?= go
CLI_NAME     := mump2p
BUILD_DIR    := dist

DOMAIN       ?= ""
CLIENT_ID    ?= ""
AUDIENCE     ?= optimum-login
SERVICE_URL  ?= http://us1-proxy.getoptimum.io:8080

LD_FLAGS := -X github.com/getoptimum/mump2p-cli/internal/config.Domain=$(DOMAIN) \
            -X github.com/getoptimum/mump2p-cli/internal/config.ClientID=$(CLIENT_ID) \
            -X github.com/getoptimum/mump2p-cli/internal/config.Audience=$(AUDIENCE) \
            -X github.com/getoptimum/mump2p-cli/internal/config.ServiceURL=$(SERVICE_URL)

.PHONY: all build run clean test lint tag release print-cli-name coverage

all: lint build

lint:
	golangci-lint run ./...

build:
	GOOS=darwin GOARCH=amd64 $(GO_BIN) build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(CLI_NAME)-mac .
	GOOS=linux GOARCH=amd64 $(GO_BIN) build -ldflags="$(LD_FLAGS)" -o $(BUILD_DIR)/$(CLI_NAME)-linux .

print-cli-name:
	@echo "$(CLI_NAME)"

release: build
	@tag=$$(git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0); \
	echo "Creating release for $$tag"; \
	mkdir -p $(BUILD_DIR); \
	gh release create "$$tag" \
		--title "Release $$tag" \
		--notes "Release notes for $$tag" \
		$(BUILD_DIR)/$(CLI_NAME)-mac \
		$(BUILD_DIR)/$(CLI_NAME)-linux
		
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
	
run: build
	./$(CLI_NAME) --config=$(CONFIG_PATH)

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

test:
	$(GO_BIN) test ./... -v -count=1

coverage:
	$(GO_BIN) test ./... -coverprofile=coverage.out -v -count=1
	$(GO_BIN) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
