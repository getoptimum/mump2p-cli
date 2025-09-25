# mump2p CLI - Complete User Guide

*This guide assumes you've completed the [Quick Start](../README.md#quick-start) from the README and are ready to explore advanced features, detailed configuration, and best practices.*

## What You'll Learn

After completing the README's quick start, this guide will teach you:

- **Authentication Management**: Token management, refresh, and troubleshooting
- **Development Mode**: Testing without authentication using `--disable-auth` flag
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

By default, authentication tokens are stored in `~/.mump2p/auth.yml`. For production deployments, security requirements, or non-root users, you can customize this location:

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

### Development/Testing Mode

For development and testing scenarios, you can bypass authentication entirely using the `--disable-auth` flag:

```sh
# All commands work without login (requires --service-url for network operations)
./mump2p --disable-auth whoami
./mump2p --disable-auth publish --topic=test --message="Hello" --service-url="http://34.146.222.111:8080"
./mump2p --disable-auth subscribe --topic=test --service-url="http://34.146.222.111:8080"
./mump2p --disable-auth list --service-url="http://34.146.222.111:8080"
./mump2p --disable-auth usage
```

**When using `--disable-auth`:**
- Uses mock client ID (`mock-client-id`)
- Unlimited rate limits for testing
- All functionality works without authentication
- **Requires `--service-url` for network operations** (publish, subscribe, list)
- Perfect for development and testing

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

*You've already tried basic topic subscription from the README. This section covers advanced options, protocols, and configuration.*

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
./mump2p subscribe --topic=alerts --webhook="https://discord.com/api/webhooks/123456789/abcdef" --webhook-schema='{"content":"{{.Message}}"}'
```
- Messages are formatted as: `{"content": "your message"}`

**Slack Webhooks:**
```sh
./mump2p subscribe --topic=notifications --webhook="https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX" --webhook-schema='{"text":"{{.Message}}"}'
```
- Messages are formatted as: `{"text": "your message"}`

**Telegram Webhooks:**
```sh
./mump2p subscribe --topic=alerts --webhook="https://api.telegram.org/bot<BOT_TOKEN>/sendMessage" --webhook-schema='{"chat_id":"<CHAT_ID>","text":"{{.Message}}"}'
```
- Messages are formatted as: `{"chat_id": "123456789", "text": "your message"}`
- Requires bot token from @BotFather and chat ID from getUpdates API

**Complex JSON Templates:**
```sh
./mump2p subscribe --topic=logs --webhook="https://your-server.com/webhook" --webhook-schema='{"message":"{{.Message}}","timestamp":"{{.Timestamp}}","topic":"{{.Topic}}","client":"{{.ClientID}}"}'
```
- Messages are formatted with multiple fields and metadata

**Raw Messages (No Schema):**
```sh
./mump2p subscribe --topic=logs --webhook="https://webhook.site/your-unique-id"
```
- Messages are sent as raw content (no JSON formatting)
- Used for custom endpoints or testing services

**Example Output:**
```text
Forwarding messages to webhook (custom schema): https://discord.com/api/webhooks/123456789/abcdef
Forwarding messages to webhook (raw format): https://webhook.site/your-unique-id
```

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

## Managing Topics

### List Your Active Topics

To see all topics you're currently subscribed to:

```sh
./mump2p list-topics
```

This will display:

- Your client ID
- Total number of active topics
- List of all subscribed topics with numbering
- Helpful guidance if no topics are found

**Example output with topics:**
```text
📋 Subscribed Topics for Client: google-oauth2|116937893938826513819
═══════════════════════════════════════════════════════════════
   Total Topics: 3

   1. test-topic-1
   2. demo-topic
   3. news-updates
═══════════════════════════════════════════════════════════════
```

**Example output with no topics:**
```text
📋 Subscribed Topics for Client: google-oauth2|116937893938826513819
═══════════════════════════════════════════════════════════════
   No active topics found.
   Use './mump2p subscribe --topic=<topic-name>' to subscribe to a topic.
═══════════════════════════════════════════════════════════════
```

### List Topics from Different Proxy

You can check your topics on a specific proxy server:

```sh
./mump2p list-topics --service-url="http://35.221.118.95:8080"
```

**Example output:**
```text
Using custom service URL: http://35.221.118.95:8080

📋 Subscribed Topics for Client: google-oauth2|116937893938826513819
═══════════════════════════════════════════════════════════════
   Total Topics: 2

   1. production-topic
   2. monitoring-topic
═══════════════════════════════════════════════════════════════
```

**Note:** Each proxy server maintains separate topic states, so you may have different topics on different proxies.

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

## Debug Mode

The `--debug` flag provides detailed timing and proxy information for troubleshooting and performance analysis. When enabled, it shows:

- **Message timestamps**: Send and receive times in nanoseconds
- **Proxy IP addresses**: Source and destination proxy information
- **Message metadata**: Size, hash, and protocol information
- **Message numbering**: Sequential numbering for received messages

### Basic Debug Usage

```sh
# Debug publish operations
./mump2p --debug publish --topic=test-topic --message='Hello World'

# Debug subscribe operations
./mump2p --debug subscribe --topic=test-topic

# Debug with gRPC
./mump2p --debug publish --topic=test-topic --message='Hello World' --grpc
./mump2p --debug subscribe --topic=test-topic --grpc
```

### Debug Output Format

**Publish Debug Output:**
```text
Publish: sender_info:34.146.222.111, [send_time, size]:[1757606701424811000, 2010] topic:test-topic msg_hash:4bbac12f protocol:HTTP
```

**Subscribe Debug Output:**
```text
Recv: [1] receiver_addr:34.146.222.111 [recv_time, size]:[1757606701424811000, 2082] sender_addr:34.146.222.111 [send_time, size]:[1757606700160514000, 2009] topic:test-topic hash:8da69366 protocol:WebSocket
```

### Debug Information Explained

- **sender_info/receiver_addr**: IP address of the proxy handling the message
- **send_time/recv_time**: Unix timestamp in nanoseconds when message was sent/received
- **size**: Message size in bytes (includes debug prefix for received messages)
- **msg_hash/hash**: First 8 characters of SHA256 hash for message identification
- **protocol**: Communication protocol used (HTTP, gRPC, or WebSocket)
- **[n]**: Sequential message number for received messages

### Load Testing with Debug Mode (Blast Testing)

Debug mode is particularly useful for load testing and performance analysis. You can send multiple messages rapidly to measure throughput and latency:

**Basic Blast Testing:**
```sh
# Terminal 1: Subscribe with debug mode
./mump2p --debug subscribe --topic=load-test --service-url="http://34.146.222.111:8080"

# Terminal 2: Send multiple messages rapidly
for i in {1..50}; do
  ./mump2p --debug publish --topic=load-test --message="Test message $i" --service-url="http://34.146.222.111:8080"
done
```

**Advanced Blast Testing with gRPC:**
```sh
# Terminal 1: Subscribe via gRPC with debug mode
./mump2p --debug subscribe --topic=grpc-load-test --grpc --service-url="http://34.146.222.111:8080"

# Terminal 2: Send 500 messages via gRPC
for i in {1..500}; do
  ./mump2p --debug publish --topic=grpc-load-test --message="GRPC test message $i" --grpc --service-url="http://34.146.222.111:8080"
done
```

**Cross-Proxy Blast Testing:**
```sh
# Terminal 1: Subscribe on one proxy
./mump2p --debug subscribe --topic=cross-proxy-test --service-url="http://34.146.222.111:8080"

# Terminal 2: Publish from different proxy (use a working proxy URL)
for i in {1..100}; do
  ./mump2p --debug publish --topic=cross-proxy-test --message="Cross-proxy message $i" --service-url="http://34.146.222.111:8080"
done
```

**Analyzing Blast Test Results:**

The debug output provides valuable metrics:
- **Throughput**: Count messages per second by analyzing timestamps
- **Latency**: Calculate `recv_time - send_time` for each message
- **Message Integrity**: Verify hashes match between send and receive
- **Proxy Performance**: Compare different proxy servers under load

**Example Blast Test Output Analysis:**
```text
# Sending 10 messages rapidly
Publish: sender_info:34.146.222.111, [send_time, size]:[1757606701424811000, 2010] topic:load-test msg_hash:4bbac12f protocol:HTTP
Publish: sender_info:34.146.222.111, [send_time, size]:[1757606701424812000, 2010] topic:load-test msg_hash:5ccbd23g protocol:HTTP
...

# Receiving messages with timing
Recv: [1] receiver_addr:34.146.222.111 [recv_time, size]:[1757606701424811000, 2082] sender_addr:34.146.222.111 [send_time, size]:[1757606701424811000, 2009] topic:load-test hash:4bbac12f protocol:WebSocket
Recv: [2] receiver_addr:34.146.222.111 [recv_time, size]:[1757606701424812000, 2082] sender_addr:34.146.222.111 [send_time, size]:[1757606701424812000, 2009] topic:load-test hash:5ccbd23g protocol:WebSocket
```

### Use Cases for Debug Mode

- **Performance Analysis**: Measure message latency and throughput
- **Troubleshooting**: Identify proxy routing and timing issues
- **Message Tracking**: Verify message integrity using hashes
- **Cross-Proxy Testing**: Monitor message flow between different proxy servers
- **Load Testing**: Analyze performance under high message volumes
- **Blast Testing**: Rapid message sending for stress testing and performance benchmarking

## Tips for Effective Use

1. **Topic Names:** Choose descriptive and unique topic names to avoid message conflicts
2. **Message Size:** Be aware of your maximum message size limit when publishing files
3. **Token Refresh:** For long-running operations, refresh your token before it expires
4. **Topic Management:** Use `./mump2p list-topics` to check your active topics and avoid duplicate topic subscriptions
5. **Persistent Subscriptions:** Use the --persist option when you need a record of messages
6. **Webhook Reliability:** Increase the queue size for high-volume topics to prevent message drops
7. **gRPC Performance:** Use `--grpc` flag for high-throughput scenarios and better performance
8. **Health Monitoring:** Check proxy health with `./mump2p health` before long operations
9. **Multi-Proxy Usage:** Remember that each proxy server maintains separate topic states - use `./mump2p list-topics --service-url=<url>` to check topics on specific proxies
10. **Debug Analysis:** Use `--debug` flag for performance monitoring and troubleshooting message flow issues

## Troubleshooting

For common setup and usage issues, see the [FAQ section in the README](../README.md#faq---common-issues--troubleshooting).

**Advanced troubleshooting:**

- **Authentication Errors:** Run `./mump2p whoami` to check token status, and `./mump2p login` to re-authenticate
- **Rate Limit Errors:** Use `./mump2p usage` to check your current usage against limits
- **Topic Issues:** 
  - Use `./mump2p list-topics` to verify your active topics
  - Check topics on different proxies with `./mump2p list-topics --service-url=<url>`
  - Remember that topics persist across logout/login sessions
- **Connection Issues:** 
  - Verify your internet connection and firewall settings
  - Check proxy server health with `./mump2p health`
  - Try a different proxy server with `--service-url` flag
- **Proxy Health Issues:** Use `./mump2p health` to check system metrics and server status
- **Webhook Failures:** 
  - Check that your webhook endpoint is accessible and properly configured to accept POST requests
  - For Discord webhooks: Use `--webhook-schema='{"content":"{{.Message}}"}'` and ensure the webhook URL is valid
  - For Slack webhooks: Use `--webhook-schema='{"text":"{{.Message}}"}'` and verify the webhook URL is correct
  - For Telegram webhooks: Use `--webhook-schema='{"chat_id":"YOUR_CHAT_ID","text":"{{.Message}}"}'` with bot token and chat ID
  - Check webhook response status codes - 400 errors usually indicate formatting issues (use appropriate schema)
  - Use [webhook.site](https://webhook.site/) for testing generic webhook endpoints
  - Define custom JSON templates with `--webhook-schema` for any webhook service
  