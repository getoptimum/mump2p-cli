#!/usr/bin/env bash
set -euo pipefail

# Load env vars if .env file exists
if [[ -f .env ]]; then
  # shellcheck disable=SC2046
  export $(grep -v '^#' .env | xargs)
fi

if [[ "${DEBUG:-}" == "true" ]]; then
  set -x
fi

err() {
  echo "[e2e] $*" >&2
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    err "environment variable $name must be set"
    exit 1
  fi
}

require_env SERVICE_URL

TOKEN_FILE="${MUMP2P_E2E_TOKEN_PATH:-}"
if [[ -z "$TOKEN_FILE" ]]; then
  if [[ -n "${MUMP2P_E2E_TOKEN_B64:-}" ]]; then
    TOKEN_TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TOKEN_TMP_DIR"' EXIT
    TOKEN_FILE="$TOKEN_TMP_DIR/auth.yml"
    if base64 --help >/dev/null 2>&1; then
      echo "${MUMP2P_E2E_TOKEN_B64}" | base64 --decode >"$TOKEN_FILE"
    else
      echo "${MUMP2P_E2E_TOKEN_B64}" | base64 -d >"$TOKEN_FILE"
    fi
  else
    err "either MUMP2P_E2E_TOKEN_PATH or MUMP2P_E2E_TOKEN_B64 must be provided"
    exit 1
  fi
fi

if [[ ! -f "$TOKEN_FILE" ]]; then
  err "token file $TOKEN_FILE does not exist"
  exit 1
fi

export MUMP2P_AUTH_PATH="$TOKEN_FILE"

host_os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$host_os" in
  linux)
    cli_bin_default="dist/mump2p-linux"
    ;;
  darwin)
    cli_bin_default="dist/mump2p-mac"
    ;;
  *)
    err "unsupported host OS: $host_os"
    exit 1
    ;;
 esac

CLI_BINARY="${MUMP2P_E2E_CLI_BINARY:-$cli_bin_default}"

if [[ ! -x "$CLI_BINARY" ]]; then
  err "cli binary $CLI_BINARY is missing; attempting to build"
  require_env AUTH_DOMAIN
  require_env AUTH_CLIENT_ID
  require_env AUTH_AUDIENCE
  make build-local \
    DOMAIN="$AUTH_DOMAIN" \
    CLIENT_ID="$AUTH_CLIENT_ID" \
    AUDIENCE="$AUTH_AUDIENCE" \
    SERVICE_URL="$SERVICE_URL"
fi

if [[ ! -x "$CLI_BINARY" ]]; then
  err "unable to locate CLI binary at $CLI_BINARY"
  exit 1
fi

TOPIC="${MUMP2P_E2E_TOPIC:-optimum-e2e-$(date +%s)}"
MESSAGE="${MUMP2P_E2E_MESSAGE:-hello from cli e2e $(date -u +%FT%TZ)}"

err "running health check"
"$CLI_BINARY" health --service-url="$SERVICE_URL"

err "verifying authentication"
"$CLI_BINARY" whoami >/dev/null

# NOTE: subscribe command is not bash friendly, so need to rewrite using go
#err "publishing test message"
#"$CLI_BINARY" publish --topic="$TOPIC" --message="$MESSAGE" --service-url="$SERVICE_URL"
#
#err "fetching topic list"
#"$CLI_BINARY" list --service-url="$SERVICE_URL" >/dev/null

#err "displaying usage"
#"$CLI_BINARY" usage >/dev/null

err "e2e smoke test finished successfully"