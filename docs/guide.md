# mump2p CLI - Complete User Guide

*This guide assumes you've completed the [Quick Start](../README.md) from the README and are ready to explore advanced features, detailed configuration, and best practices.*

## What You'll Learn

After completing the README's quick start, this guide will teach you:

- **Authentication Management**: Token management, refresh, and troubleshooting
- **Development Mode**: Testing without authentication using `--disable-auth` flag
- **Service Configuration**: Using different proxy servers and custom URLs
- **Direct P2P**: How messages flow directly to nodes via ADR-0002
- **Advanced Features**: Message persistence, webhooks, and monitoring
- **Production Best Practices**: Performance optimization and troubleshooting

---

## Prerequisites

Before following this guide, ensure you have:
- ✅ **Installed the CLI** via the install script or manual download
- ✅ **Completed authentication** with `mump2p login`
- ✅ **Tested basic publish/subscribe** from the README examples

*If you haven't done these steps yet, please start with the [README](../README.md) first.*

---

## Authentication

*You should already be authenticated from the README quick start. This section covers authentication management and troubleshooting.*

### Managing Your Session

Check your current authentication status and token details:

```sh
mump2p whoami
```

This displays:
- Your client ID and email
- Token expiration time (24 hours from login)
- Token validity status
- Rate limits and quotas for your account

### Refresh Token

If your token is about to expire, you can refresh it:

```sh
mump2p refresh
```

### Custom Authentication File Location

By default, authentication tokens are stored in `~/.mump2p/auth.yml`. For production deployments, security requirements, or non-root users, you can customize this location:

```sh
# Use custom authentication file path
mump2p --auth-path /opt/mump2p/auth/token.yml login

# All subsequent commands will use the same custom path
mump2p --auth-path /opt/mump2p/auth/token.yml publish --topic=demo --message="Hello"
mump2p --auth-path /opt/mump2p/auth/token.yml logout
```

**Environment Variable Support:**
```sh
# Set via environment variable (applies to all commands)
export MUMP2P_AUTH_PATH="/opt/mump2p/auth/token.yml"
mump2p login
mump2p publish --topic=demo --message="Hello"
```

**Use Cases:**
- **Security**: Store auth files in secure, restricted directories
- **Deployment Automation**: Use with Ansible, Terraform without root permissions
- **Multi-user Environments**: Separate auth files per user/service
- **Container Deployments**: Mount auth files from persistent volumes

**Important Notes:**
- The directory will be created automatically if it doesn't exist
- Rate limiting usage files will be stored in the same directory
- Ensure the user has write permissions to the specified directory

### Development/Testing Mode

For development and testing scenarios, you can bypass authentication entirely using the `--disable-auth` flag:

```sh
# All commands work without login (requires --client-id and --service-url)
mump2p --disable-auth --client-id="my-test-client" whoami
mump2p --disable-auth --client-id="my-test-client" publish --topic=test --message="Hello" --service-url="http://us1-proxy.getoptimum.io:8080"
mump2p --disable-auth --client-id="my-test-client" subscribe --topic=test --service-url="http://us1-proxy.getoptimum.io:8080"
mump2p --disable-auth --client-id="my-test-client" list-topics --service-url="http://us1-proxy.getoptimum.io:8080"
mump2p --disable-auth usage

# Combine with debug mode
mump2p --disable-auth --client-id="my-test-client" --debug publish --topic=test --message="Hello" --service-url="http://us1-proxy.getoptimum.io:8080"
```

**When using `--disable-auth`:**
- **Must provide `--client-id` flag** with your chosen client ID
- No rate limits enforced (bypasses all quotas)
- No usage tracking
- All functionality works without authentication
- **Requires `--service-url` for network operations** (publish, subscribe, list-topics)

### Logout

To remove your stored authentication token:

```sh
mump2p logout
```

---

## Service URL Configuration

*The README used the default proxy. This section shows how to configure different proxy servers for better performance or geographic proximity.*

The CLI connects to proxy servers for session management, then communicates directly with P2P nodes. By default, it uses `http://us1-proxy.getoptimum.io:8080`, but you can specify a different one using the `--service-url` flag.

**Available proxies:**
- `http://us1-proxy.getoptimum.io:8080`
- `http://us2-proxy.getoptimum.io:8080`
- `http://us3-proxy.getoptimum.io:8080`

**Example using a specific proxy:**
```sh
mump2p publish --topic=test --message='Hello' --service-url="http://us2-proxy.getoptimum.io:8080"
mump2p subscribe --topic=test --service-url="http://us3-proxy.getoptimum.io:8080"
```

---

## Subscribing to Messages - Deep Dive

*You've already tried basic topic subscription from the README. This section covers advanced options and configuration.*

### How Subscribe Works (ADR-0002)

1. CLI requests a session from the proxy (control plane)
2. Proxy returns a list of scored P2P nodes with JWT tickets
3. CLI connects directly to the best node via gRPC
4. Messages stream directly from the node — no proxy hop

### Basic Subscription

```sh
mump2p subscribe --topic=your-topic-name
```

By default, 3 nodes are requested for automatic failover. If the primary node fails, the CLI falls back to the next one.

### Save Messages to a File

To persist messages to a local file while subscribing:

```sh
mump2p subscribe --topic=your-topic-name --persist=/path/to/save/
```

If you provide just a directory path, messages will be saved to a file named `messages.log` in that directory.

### Forward Messages to a Webhook

To forward messages to an HTTP webhook:

```sh
mump2p subscribe --topic=your-topic-name --webhook=https://your-server.com/webhook
```

**Note: The webhook endpoint must be configured to accept POST requests.**

#### Webhook Formatting

The CLI supports flexible JSON template formatting for webhooks. You can define custom schemas using Go template syntax with available variables.

**Available Template Variables:**
- `{{.Message}}` - The message content
- `{{.Timestamp}}` - Message timestamp (RFC3339 format)
- `{{.Topic}}` - The topic name
- `{{.ClientID}}` - Sender's client ID
- `{{.MessageID}}` - Unique message identifier

**Discord Webhooks:**
```sh
mump2p subscribe --topic=alerts --webhook="https://discord.com/api/webhooks/123456789/abcdef" --webhook-schema='{"content":"{{.Message}}"}'
```

**Slack Webhooks:**
```sh
mump2p subscribe --topic=notifications --webhook="https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX" --webhook-schema='{"text":"{{.Message}}"}'
```

**Telegram Webhooks:**
```sh
mump2p subscribe --topic=alerts --webhook="https://api.telegram.org/bot<BOT_TOKEN>/sendMessage" --webhook-schema='{"chat_id":"<CHAT_ID>","text":"{{.Message}}"}'
```

**Raw Messages (No Schema):**
```sh
mump2p subscribe --topic=logs --webhook="https://webhook.site/your-unique-id"
```

#### Advanced Webhook Options

For more control over webhook behavior:

```sh
mump2p subscribe --topic=your-topic-name \
  --webhook=https://your-server.com/webhook \
  --webhook-queue-size=200 \
  --webhook-timeout=5
```

Options:

- `--webhook-queue-size`: Maximum number of messages to queue before dropping (default: `100`)
- `--webhook-timeout`: Timeout in seconds for each webhook POST request (default: `3`)

### Combine Persistence and Webhook

You can both save messages locally and forward them to a webhook:

```sh
mump2p subscribe --topic=your-topic-name \
  --persist=/path/to/messages.log \
  --webhook=https://your-server.com/webhook
```

---

## Publishing Messages - Deep Dive

*You've tried basic publishing from the README. This section covers advanced publishing options and file handling.*

### Inline Publishing

```sh
mump2p publish --topic=your-topic-name --message='Your message content'
```

### Publish a File

To publish the contents of a file:

```sh
mump2p publish --topic=your-topic-name --file=/path/to/your/file.json
```

Rate limits will be automatically applied based on your authentication token.

---

## Managing Topics

### List Your Active Topics

To see all topics you're currently subscribed to:

```sh
mump2p list-topics
```

### List Topics from Different Proxy

You can check your topics on a specific proxy server:

```sh
mump2p list-topics --service-url="http://us2-proxy.getoptimum.io:8080"
```

**Note:** Each proxy server maintains separate topic states, so you may have different topics on different proxies.

## Checking Usage and Limits

To see your current usage statistics and rate limits:

```sh
mump2p usage
```

## Tracer Dashboard

Interactive real-time dashboard showing network metrics, message statistics, and latency data.

```sh
mump2p tracer dashboard
```

**Options:**
- `--window`: Time window for metrics (default: `10s`)
- `--topic`: Topic for auto-publishing demo messages (default: `demo`)
- `--count`: Number of messages to auto-publish (default: `60`)
- `--interval-ms`: Interval between published messages in ms (default: `500`)

Press `q` or `Ctrl+C` to exit.

## Health Monitoring

### Check Proxy Server Health

```sh
mump2p health
```

### Check Health of Specific Proxy

```sh
mump2p health --service-url="http://us2-proxy.getoptimum.io:8080"
```

---

## Debug Mode

The `--debug` flag provides detailed session, node, and timing information for troubleshooting and performance analysis. When enabled, it shows:

- **Session details**: Session ID, proxy URL, session creation timing
- **Node selection**: Node addresses, regions, scores, connection attempts
- **Message metadata**: Timestamps, size, hash, protocol, P2P peer paths
- **Timing breakdown**: Session vs publish/subscribe timing

### Basic Debug Usage

```sh
# Debug publish
mump2p --debug publish --topic=test-topic --message='Hello World'

# Debug subscribe
mump2p --debug subscribe --topic=test-topic
```

### Load Testing with Debug Mode

Debug mode is useful for load testing and performance analysis:

```sh
# Terminal 1: Subscribe with debug mode
mump2p --debug subscribe --topic=load-test

# Terminal 2: Send multiple messages rapidly
for i in {1..50}; do
  mump2p --debug publish --topic=load-test --message="Test message $i"
done
```

---

## Tips for Effective Use

1. **Topic Names:** Choose descriptive and unique topic names to avoid message conflicts
2. **Message Size:** Be aware of your maximum message size limit when publishing files
3. **Token Refresh:** For long-running operations, refresh your token before it expires
4. **Topic Management:** Use `mump2p list-topics` to check your active topics
5. **Persistent Subscriptions:** Use the `--persist` option when you need a record of messages
6. **Webhook Reliability:** Increase the queue size for high-volume topics to prevent message drops
7. **Failover:** Use `--expose-amount` to control how many backup nodes are available (subscribe defaults to 3)
8. **Health Monitoring:** Check proxy health with `mump2p health` before long operations
9. **Debug Analysis:** Use `--debug` flag for performance monitoring and troubleshooting message flow issues

## Troubleshooting

- **Authentication Errors:** Run `mump2p whoami` to check token status, and `mump2p login` to re-authenticate
- **Rate Limit Errors:** Use `mump2p usage` to check your current usage against limits
- **Topic Issues:** Use `mump2p list-topics` to verify your active topics
- **Connection Issues:** Check proxy server health with `mump2p health`, try a different proxy with `--service-url`
- **Webhook Failures:** Check that your webhook endpoint is accessible and properly configured to accept POST requests
