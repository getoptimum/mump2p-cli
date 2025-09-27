# End-to-End Smoke Testing for `mump2p-cli`

This guide describes how to exercise the CLI binary against a running Optimum proxy (`SERVICE_URL`) before publishing a release.
The goal is to run an automated set of tests in the release pipeline.
The checks only pass when both **authentication** and the **proxy** are available.

The walkthrough is split into three phases:

1. **Prepare credentials** that let the script authenticate non-interactively.
2. **Run the script locally** to verify configuration.
3. **Invoke the script in CI** against the release binary.

---

## Overview

1. Provide the CLI with the same Auth0 configuration used to build the release binary:

    * `AUTH_DOMAIN`
    * `AUTH_CLIENT_ID`
    * `AUTH_AUDIENCE`
    * `SERVICE_URL` (proxy endpoint to test against)
2. Supply token (YAML encoded `StoredToken`) via CI secrets so the test can authenticate uninteractively
3. Execute `scripts/e2e-smoke.sh`. The script runs the following checks:

    * Health probe:

      ```bash
      mump2p health --service-url "$SERVICE_URL"
      ```
    * Token verification:

      ```bash
      mump2p whoami
      ```
    * Publish over HTTP:

      ```bash
      mump2p publish --service-url "$SERVICE_URL"
      ```
    * Topic listing:

      ```bash
      mump2p list --service-url "$SERVICE_URL"
      ```
    * Local usage accounting:

      ```bash
      mump2p usage
      ```
4. If every command exits successfully, the script exits with `0`. The release pipeline can then continue with publishing (planned)

---

## 1. Preparing Credentials

The script needs a token that looks like the YAML file persisted in `~/.mump2p/auth.yml`:

```yaml
# example structure
---
token: eyJhbGciOi...
refresh_token: c29tZS1yZWZyZXNoLXRva2Vu
expires_at: 2024-11-13T10:15:19Z
```

To avoid handling multi-line secrets directly:
* Encode the YAML payload as base64 and store it as a CI secret (e.g. `MUMP2P_E2E_TOKEN_B64`).
> **Tip:** When running locally you can generate the base64 payload with:
>
> ```bash
> base64 < ~/.mump2p/auth.yml | tr -d '\n'
> ```

### Required Environment Variables

| Variable                                          | Description                                               |
| ------------------------------------------------- | --------------------------------------------------------- |
| `SERVICE_URL`                                     | Remote proxy base URL (e.g. `https://proxy.example.com`). |
| `AUTH_DOMAIN`                                     | Auth0 domain used to issue tokens for the CLI.            |
| `AUTH_CLIENT_ID`                                  | Auth0 client id corresponding to the CLI application.     |
| `AUTH_AUDIENCE`                                   | Auth0 audience expected by the proxy.                     |
| `MUMP2P_E2E_TOKEN_B64` or `MUMP2P_E2E_TOKEN_PATH` | Location of the stored token.                             |

---

## 2. Running Locally

1. Export the configuration:

   ```bash
   export SERVICE_URL="https://proxy.example.com"
   export AUTH_DOMAIN="domain"
   export AUTH_CLIENT_ID="123..."
   export AUTH_AUDIENCE="optimum-login"
   # instead of command its output
   export MUMP2P_E2E_TOKEN_B64="$(base64 < ~/.mump2p/auth.yml | tr -d '\n')" 
   ```

2. (Optional) Point the test at a pre-built CLI artifact:

   ```bash
   export MUMP2P_E2E_CLI_BINARY="/path/to/mump2p"
   ```

3. Run the smoke test:

   ```bash
   ./scripts/e2e-smoke.sh
   ```

The script will build the CLI with the supplied flags when `MUMP2P_E2E_CLI_BINARY` is absent.
A successful run logs each check and exits with status `0`.

### Optional overrides

* `MUMP2P_E2E_TOPIC` – topic name used for publishing (defaults to a timestamp).
* `MUMP2P_E2E_MESSAGE` – message content for the publish step.
* `DEBUG=true` – enable verbose shell output for troubleshooting.

---

## 3. Integrating with the Release Pipeline

1. **Add job** after building binary

2. **Inject secrets** 

   | Variable                                                  | Purpose                                    |
      |-----------------------------------------------------------| ------------------------------------------ |
   | `SERVICE_URL`                                             | Remote proxy base URL.                     |
   | `AUTH_DOMAIN`                                             | Auth0 domain        |
   | `AUTH_CLIENT_ID`                                          | Auth0 client id    |
   | `AUTH_AUDIENCE`                                           | Auth0 audience used     |
   | `MUMP2P_E2E_TOKEN_B64` or `MUMP2P_E2E_TOKEN_PATH` (local) | Stored token for automated authentication. |
   | `MUMP2P_E2E_CLI_BINARY` *(optional)*                      | Path to the binary you want to test.       |

3. **Run the script** with a single command:

   ```bash
   ./scripts/e2e-smoke.sh
   ```

4. **Fail the release** when the script exits with a non-zero status. This prevents publishing when either authentication or the proxy is down.

