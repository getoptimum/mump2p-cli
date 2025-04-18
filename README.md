# mump2p — OptimumP2P CLI

`mump2p` is the command-line interface for interacting with [OptimumP2P](https://github.com/getoptimum/optimum-p2p) — a high-performance RLNC-enhanced pubsub protocol.

It supports authenticated publishing, subscribing, rate-limited usage tracking, and JWT session management.

---

This CLI allows you to:

- [x] Publish messages to topics
- [ ] Subscribe to real-time message streams
- [x] JWT-based login/logout and token refresh
- [x] Local rate-limiting (publish count, quota, max size)
- [x] Usage statistics reporting

---

## Installation

```sh
git clone https://github.com/getoptimum/optcli
cd optcli
make build
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
TODO::
```

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
- [ ] Subscribe Message
- [x] JWT-based login/logout/refresh
- [x] Token-based rate limits
- [x] Usage tracking (usage command)
- [ ] Real-time stream mode
- [ ] `follow` mode for replaying logs
