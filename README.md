# 🧪 mump2p — CLI

`mump2p` is the command-line tool to interact with [OptimumP2P](https://github.com/getoptimum/optimum-p2p) — a high-performance RLNC-enhanced pubsub protocol.

This CLI allows you to:

- [ ] Register nodes and keys
- [x] Publish messages to topics
- [x] Subscribe to topics
- [ ] live message stream

---

## Installation

```sh
git clone https://github.com/getoptimum/optcli
cd optcli
make build
```

## Usage

### Register a node/key to a topic

```sh
TODO
```

### Publish Message

```sh
./mump2p publish \
  --topic=test-topic \
  --message="new block 1234" \
  --algorithm=optimump2p \
  --config=app_conf.yml
```

### Subscribe

```sh
./mump2p subscribe \
  --topic=data \
  --algorithm=optimump2p \
  --config=app_conf.yml
```

## Roadmap

- [x] Publish Message
- [x] Subscribe Message
- [ ] Register Node/Key
- [ ] Keyring support for signing
- [ ] JWT Session auth
- [ ] Message Listener
