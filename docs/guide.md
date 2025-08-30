# OptimumP2P CLI - Complete User Guide

*This guide assumes you've completed the [Quick Start](../README.md#quick-start) from the README and are ready to explore advanced features, detailed configuration, and best practices.*

## What You'll Learn

After completing the README's quick start, this guide will teach you:

- **Authentication Management**: Token management, refresh, and troubleshooting
- **Service Configuration**: Using different proxy servers and custom URLs  
- **Protocol Deep Dive**: When to use HTTP/WebSocket vs gRPC
- **Advanced Features**: Message persistence, webhooks, and monitoring
- **Production Best Practices**: Performance optimization and troubleshooting

---

## Prerequisites

Before following this guide, ensure you have:
- ✅ **Installed the CLI** via the [install script](../README.md#1-installation) or manual download
- ✅ **Completed authentication** with `./mump2p login`
- ✅ **Tested basic publish/subscribe** from the README examples

*If you haven't done these steps yet, please start with the [README Quick Start](../README.md#quick-start) first.*

---

## Authentication

*You should already be authenticated from the README quick start. This section covers authentication management and troubleshooting.*

### Managing Your Session

Check your current authentication status and token details:

```sh
./mump2p whoami
```

This displays:
- Your client ID and email
- Token expiration time (24 hours from login)
- Token validity status  
- Rate limits and quotas for your account

**Example output:**
```text
Authentication Status:
----------------------
Client ID: google-oauth2|100677750055416883405
Expires: 24 Sep 25 20:44 IST
Valid for: 706h53m0s
Is Active:  true

Rate Limits:
------------
Publish Rate:  1000 per hour
Publish Rate:  8 per second
Max Message Size:  4.00 MB
Daily Quota:       5120.00 MB
```

### Refresh Token

If your token is about to expire, you can refresh it:

```sh
./mump2p refresh
```

### Custom Authentication File Location

By default, authentication tokens are stored in `~/.optimum/auth.yml`. For production deployments, security requirements, or non-root users, you can customize this location:

```sh
# Use custom authentication file path
./mump2p --auth-path /opt/mump2p/auth/token.yml login

# All subsequent commands will use the same custom path
./mump2p --auth-path /opt/mump2p/auth/token.yml publish --topic=demo --message="Hello"
./mump2p --auth-path /opt/mump2p/auth/token.yml logout
```

**Environment Variable Support:**
```sh
# Set via environment variable (applies to all commands)
export MUMP2P_AUTH_PATH="/opt/mump2p/auth/token.yml"
./mump2p login
./mump2p publish --topic=demo --message="Hello"
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

### Logout

To remove your stored authentication token:

```sh
./mump2p logout
```

---

## Service URL Configuration

*The README used the default proxy. This section shows how to configure different proxy servers for better performance or geographic proximity.*

The CLI connects to different proxy servers around the world. By default, it uses the first available proxy, but you can specify a different one using the `--service-url` flag for better performance or closer geographic location.

For a complete list of available proxies and their locations, see: [Available Service URLs](../README.md#available-service-urls) in the README.

**Example using a specific proxy:**
```sh
./mump2p publish --topic=test --message='Hello' --service-url="http://35.221.118.95:8080"
./mump2p subscribe --topic=test --service-url="http://34.142.205.26:8080"
```

**Example output when using custom service URL:**
```text
Using custom service URL: http://34.142.205.26:8080
✅ Published via HTTP inline message
{"message_id":"f5f51132c83da5a0209d6348bffe7eb1dafc91544e9240b98ac2c8e6da25c410","status":"published","topic":"demo"}
```

---

## Subscribing to Messages - Deep Dive

*You've already tried basic subscription from the README. This section covers advanced options, protocols, and configuration.*

### Understanding WebSocket vs gRPC

The README showed you both protocols. Here's when to use each:

**WebSocket (Default)** - Good for:
- Getting started and testing
- Lower resource usage
- Standard web-compatible streaming

**gRPC** - Best for:
- High-throughput scenarios (1000+ messages/sec)
- Production environments
- Better performance and reliability

### Basic WebSocket Subscription

You've seen this from the README:

```sh
./mump2p subscribe --topic=your-topic-name
```

This will open a WebSocket connection and display incoming messages in real-time. Press `Ctrl+C` to exit.

**Example output (WebSocket):**
```text
Using custom service URL: http://34.142.205.26:8080
Sending HTTP POST subscription request...
HTTP POST subscription successful: {"client":"google-oauth2|100677750055416883405","status":"subscribed"}
Opening WebSocket connection...
Listening for messages on topic 'demo'... Press Ctrl+C to exit
```

### gRPC Subscription (Advanced)

From the README, you saw the `--grpc` flag. Here's the detailed breakdown:

```sh
./mump2p subscribe --topic=your-topic-name --grpc
```

**Example output:**
```text
Using custom service URL: http://34.142.205.26:8080
Sending HTTP POST subscription request...
HTTP POST subscription successful: {"client":"google-oauth2|100677750055416883405","status":"subscribed"}
Listening for messages on topic 'demo' via gRPC... Press Ctrl+C to exit
```

gRPC provides:
- **Better performance** than WebSocket for high-throughput scenarios
- **Binary protocol** with smaller message overhead  
- **Bidirectional streaming** support
- **Built-in retry and error handling**

### Save Messages to a File

To persist messages to a local file while subscribing:

```sh
./mump2p subscribe --topic=your-topic-name --persist=/path/to/save/
```

With gRPC:
```sh
./mump2p subscribe --topic=your-topic-name --persist=/path/to/save/ --grpc
```

If you provide just a directory path, messages will be saved to a file named `messages.log` in that directory.

### Forward Messages to a Webhook

To forward messages to an HTTP webhook:

```sh
./mump2p subscribe --topic=your-topic-name --webhook=https://your-server.com/webhook
```

With gRPC:
```sh
./mump2p subscribe --topic=your-topic-name --webhook=https://your-server.com/webhook --grpc
```

**Note: The webhook endpoint must be configured to accept POST requests.**

#### Advanced Webhook Options

For more control over webhook behavior:

```sh
./mump2p subscribe --topic=your-topic-name \
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
./mump2p subscribe --topic=your-topic-name \
  --persist=/path/to/messages.log \
  --webhook=https://your-server.com/webhook
```

With gRPC:
```sh
./mump2p subscribe --topic=your-topic-name \
  --persist=/path/to/messages.log \
  --webhook=https://your-server.com/webhook \
  --grpc
```

---

## Publishing Messages - Deep Dive

*You've tried basic publishing from the README. This section covers advanced publishing options, protocols, and file handling.*

### HTTP Publishing (From README)

You've already used this basic command:

```sh
./mump2p publish --topic=your-topic-name --message='Your message content'
```

**Example output:**
```text
✅ Published via HTTP inline message
{"message_id":"9cbf2612dd4371d154babad4e7b88e1f98b34cdf740283a406600f0959bdffff","status":"published","topic":"demo"}
```

### gRPC Publishing (Advanced)

From the README, you saw the `--grpc` flag for publishing. Here's when and how to use it:

```sh
./mump2p publish --topic=your-topic-name --message='Your message content' --grpc
```

**Example output:**
```text
✅ Published via gRPC inline message
```

**With custom service URL:**
```text
Using custom service URL: http://34.142.205.26:8080
✅ Published via gRPC inline message
```

### Publish a File

To publish the contents of a file (HTTP):

```sh
./mump2p publish --topic=your-topic-name --file=/path/to/your/file.json
```

To publish a file via gRPC:

```sh
./mump2p publish --topic=your-topic-name --file=/path/to/your/file.json --grpc
```

Rate limits will be automatically applied based on your authentication token.

---

## Checking Usage and Limits

To see your current usage statistics and rate limits:

```sh
./mump2p usage
```

This will display:

- Number of publish operations (per hour and per second)
- Data usage (bytes, KB, or MB depending on amount)
- Quota limits
- Time until usage counters reset
- Timestamps of your last publish and subscribe operations

## Health Monitoring

### Check Proxy Server Health

To monitor the health and system metrics of the proxy server you're connected to:

```sh
./mump2p health
```

This will display:

- **Status**: Current health status of the proxy ("ok" if healthy)
- **Memory Used**: Memory usage percentage
- **CPU Used**: CPU usage percentage  
- **Disk Used**: Disk usage percentage

**Example output:**

```text
Proxy Health Status:
-------------------
Status:      ok
Memory Used: 7.02%
CPU Used:    0.25%
Disk Used:   44.05%
```

### Check Health of Specific Proxy

You can check the health of a specific proxy server:

```sh
./mump2p health --service-url="http://35.221.118.95:8080"
```

This is useful for:
- Monitoring multiple proxy servers
- Checking proxy health before switching service URLs
- Troubleshooting connection issues
- Performance monitoring and capacity planning

---

## Tips for Effective Use

1. **Topic Names:** Choose descriptive and unique topic names to avoid message conflicts
2. **Message Size:** Be aware of your maximum message size limit when publishing files
3. **Token Refresh:** For long-running operations, refresh your token before it expires
4. **Persistent Subscriptions:** Use the --persist option when you need a record of messages
5. **Webhook Reliability:** Increase the queue size for high-volume topics to prevent message drops
6. **gRPC Performance:** Use `--grpc` flag for high-throughput scenarios and better performance
7. **Health Monitoring:** Check proxy health with `./mump2p health` before long operations

## Troubleshooting

For common setup and usage issues, see the [FAQ section in the README](../README.md#faq---common-issues--troubleshooting).

**Advanced troubleshooting:**

- **Authentication Errors:** Run `./mump2p whoami` to check token status, and `./mump2p login` to re-authenticate
- **Rate Limit Errors:** Use `./mump2p usage` to check your current usage against limits
- **Connection Issues:** 
  - Verify your internet connection and firewall settings
  - Check proxy server health with `./mump2p health`
  - Try a different proxy server with `--service-url` flag
- **Proxy Health Issues:** Use `./mump2p health` to check system metrics and server status
- **Webhook Failures:** Check that your webhook endpoint is accessible and properly configured to accept POST requests
  