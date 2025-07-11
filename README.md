# mump2p — OptimumP2P CLI

`mump2p` is the command-line interface for interacting with [OptimumP2P](https://github.com/getoptimum/optimum-p2p) — a high-performance RLNC-enhanced pubsub protocol.

It supports authenticated publishing, subscribing, rate-limited usage tracking, and JWT session management.

**[Detailed User Guide - Step-by-step instructions for all features](./docs/guide.md)**

---

This CLI allows you to:

- [x] Publish messages to topics
- [x] Subscribe to real-time message streams
- [x] JWT-based login/logout and token refresh
- [x] Local rate-limiting (publish count, quota, max size)
- [x] Usage statistics reporting
- [x] Persist messages to local storage
- [x] Forward messages to webhook endpoints (important: webhook take `POST` http method)
  
---

## Installation

```sh
git clone https://github.com/getoptimum/mump2p-cli
cd mump2p-cli
# config ldflags
export DOMAIN="xxx"
export CLIENT_ID="xxx"
export SERVICE_URL="xxx"
make build

# or
DOMAIN="xxx" CLIENT_ID="xxx" SERVICE_URL="xxx" make build

```

## Version Compatibility

**Important:** Always use the latest version binaries (currently **v0.0.1-rc3**) from the releases page. 

**Deprecated Versions:**
- ⚠️ **v0.0.1-rc2** and **v0.0.1-rc1** are deprecated and no longer supported
- Please upgrade to **v0.0.1-rc3**

## Authentication

Before publishing or subscribing, login via device flow:

```sh
./mump2p login
```

To check the current session:

```sh
./mump2p whoami
```

To refresh the session token manually:

```sh
./mump2p refresh
```

To logout:

```sh
./mump2p logout
```

## Usage

### Publish Message

```sh
./mump2p publish --topic=test-topic --message="new block 1234"
```

Message size and rate limits will be validated using the authenticated token claims. CLI do it internally.

(optional, custom endpoint)

```sh
./mump2p publish --topic=test-topic --message="new block 1234" --service-url="https://your-custom-endpoint.com"
```

### Subscribe to a Topic

```sh
./mump2p subscribe --topic=test-topic
```

Subscribe and persist messages to a local file:

```sh
./mump2p subscribe --topic=test-topic --persist=/path/to/
```

Subscribe and forward messages to a webhook:

```sh
./mump2p subscribe --topic=test-topic --webhook=https://your-server.com/webhook
```

You can combine both persistence and webhook forwarding:

```sh
./mump2p subscribe --topic=test-topic --persist=/path/to/ --webhook=https://your-server.com/webhook
```

Advanced webhook options:

```sh
./mump2p subscribe --topic=test-topic --webhook=https://your-server.com/webhook --webhook-queue-size=200 --webhook-timeout=5
```

(optional, custom endpoint)

```sh
./mump2p subscribe --topic=test-topic --service-url="https://your-custom-endpoint.com"
```

here:

- `--webhook-queue-size:` Max number of webhook messages to queue before dropping (default: 100)
- `--webhook-timeout:` Timeout in seconds for each webhook POST request (default: 3)
- `--service-url`: Optional custom service url

## Example: Subscribe to a topic using WebSocket (default)

```sh
mump2p-cli subscribe --topic my-topic
```

## Example: Subscribe to a topic using gRPC stream

```sh
mump2p-cli subscribe --topic my-topic --grpc
```

- Use `--grpc` to enable gRPC streaming subscription instead of WebSocket.
- All other flags (e.g., --persist, --webhook, --threshold) are supported in both modes.

## Check Rate Limits & Usage

```sh
./mump2p usage
```

This shows:

- Current publish count
- Daily data quota used
- Time until reset
- Token expiry info

## Roadmap

- [x] Publish Message
- [x] Subscribe Message
- [x] JWT-based login/logout/refresh
- [x] Token-based rate limits
- [x] Usage tracking (usage command)
- [x] Real-time stream mode
- [x] Message persistence
- [x] Webhook forwarding

---

## FAQ - Common Issues & Troubleshooting

### **1. Authentication & Account Issues**

#### **Error: `unauthorized_client` during login**
```
Error: device code request failed: {"error":"unauthorized_client","error_description":"Unauthorized or unknown client"}
```

**Causes:**
- Incorrect Client ID in build configuration
- Auth0 application not enabled for Device Code flow
- Wrong Domain or Audience values
- Auth0 application type incorrectly configured

**Solutions:**
- Verify Auth0 application settings
- Enable Device Code grant type in Auth0
- Use correct Domain, Client ID, and Audience values

#### **Error: `your account is inactive`**
```
Error: your account is inactive, please contact support
```

**Causes:**
- User's `is_active` flag set to `false` in Auth0
- Token issued before account activation
- Missing or incorrect `app_metadata` in Auth0

**Solutions:**
- Update user's `app_metadata.is_active` to `true` in Auth0
- Logout and login again to get new token with updated claims
- Contact admin to activate your account

### **2. Build & Configuration Issues**

#### **Error: Binary not found**
```
zsh: no such file or directory: ./mump2p
```

**Causes:**
- CLI not built yet
- Wrong binary name or path
- Binary not executable

**Solutions:**
- Run `make build` with correct environment variables
- Use correct path: `./dist/mump2p-mac`
- Make binary executable: `chmod +x dist/mump2p-mac`

#### **Error: Wrong Service URL in build**

**Causes:**
- Using localhost when should use remote URL
- Using remote URL when should use localhost
- Service URL not matching actual gateway

**Solutions:**
- Rebuild with correct SERVICE_URL
- Use `--service-url` flag to override
- Match SERVICE_URL to your actual gateway endpoint

### **3. Service URL & Connectivity Issues**

#### **Available Service URLs**

By default, the CLI uses the first gateway in the list below. You can override this using the `--service-url` flag or by rebuilding with a different `SERVICE_URL`.

| **Gateway Address** | **Location** | **URL** | **Notes** |
|---------------------|--------------|---------|-----------|
| `34.146.222.111` | Tokyo | `http://34.146.222.111:8080` | **Default** |
| `35.221.118.95` | Tokyo | `http://35.221.118.95:8080` | |
| `34.142.205.26` | Singapore | `http://34.142.205.26:8080` | |

> **Note:** More geo-locations coming soon!

**Example: Using a different gateway:**

```sh
./mump2p-mac publish --topic=example-topic --message="Hello" --service-url="http://35.221.118.95:8080"
./mump2p-mac subscribe --topic=example-topic --service-url="http://34.142.205.26:8080"
```

#### **Error: Connection refused**
```
Error: HTTP publish failed: dial tcp [::1]:8080: connect: connection refused
```

**Causes:**
- Gateway not running
- Wrong port or hostname
- Firewall blocking connection
- Service not listening on specified port

**Solutions:**
- Start gateway service
- Verify correct hostname and port
- Check `docker ps` for running containers
- Use correct service URL
- Try a different gateway from the table above

### **4. Rate Limiting & Usage Issues**

#### **Error: Rate limit exceeded**
```
Error: per-hour limit reached (100/hour)
Error: daily quota exceeded
Error: message size exceeds limit
```

**Causes:**
- Publishing too frequently
- Message too large for tier
- Daily quota exhausted
- Per-second limit hit

**Solutions:**
- Wait for rate limit reset
- Use smaller messages
- Check usage: `./mump2p usage`
- Contact admin for higher limits
- Spread out publish operations

#### **Error: Token expired**
```
Error: token has expired, please login again
```

**Causes:**
- JWT token expired (24 hours)
- Clock skew
- Token corrupted

**Solutions:**
- Refresh token: `./mump2p refresh`
- Login again: `./mump2p login`
- Check system time

### **5. Docker & Networking Issues**

#### **Error: Container name conflicts**
```
Error: Conflict. The container name "/p2pnode1" is already in use
```

**Causes:**
- Container with same name already running
- Previous container not cleaned up

**Solutions:**
- Stop and remove existing container: `docker stop <name> && docker rm <name>`
- Use different container name
- Clean up containers: `docker container prune`

#### **Error: Name resolution in Docker**
```
Error: name resolver error: produced zero addresses
```

**Causes:**
- Containers not on same Docker network
- Using container names without custom network
- Hostname not resolvable between containers

**Solutions:**
- Create custom Docker network: `docker network create optimum-net`
- Run containers on same network: `--network optimum-net`
- Use container names as hostnames in configuration

### **6. CLI Usage & Syntax Issues**

#### **Error: Missing required flags**
```
Error: required flag(s) "topic" not set
```

**Causes:**
- Forgetting required command line arguments
- Typos in flag names

**Solutions:**
- Use `--help` to see required flags
- Include all required arguments
- Check flag spelling and syntax

---

**Pro Tips for First-Time Users:**
- Always check `docker ps` and `docker logs` for container status
- Use `--help` flag liberally to understand command options
- Test authentication first with `whoami` before trying other operations
- Start with simple publish/subscribe before advanced features
- Keep gateway and CLI logs visible during troubleshooting
- Use [webhook.site](https://webhook.site/) for easy webhook testing
- Check `usage` command regularly to monitor limits
