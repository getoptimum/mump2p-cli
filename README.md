# mump2p — OptimumP2P CLI

`mump2p` is the command-line interface for interacting with [OptimumP2P](https://github.com/getoptimum/optimum-p2p) — a high-performance RLNC-enhanced pubsub protocol.

It supports authenticated publishing, subscribing, rate-limited usage tracking, and JWT session management.

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
git clone https://github.com/getoptimum/optcli
cd optcli
# config ldflags
export DOMAIN="xxx"
export CLIENT_ID="xxx"
export SERVICE_URL="xxx"
make build

# or
DOMAIN="xxx" CLIENT_ID="xxx" SERVICE_URL="xxx" make build

```

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

here:

- `--webhook-queue-size:` Max number of webhook messages to queue before dropping (default: 100)
- `--webhook-timeout:` Timeout in seconds for each webhook POST request (default: 3)

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
