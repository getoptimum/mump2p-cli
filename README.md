# mump2p CLI

CLI for interacting with the Optimum P2P network. Connects to a proxy for session management, then communicates directly with nodes for publish/subscribe.

## Authentication

### Login

```bash
mump2p login
```

```
Initiating authentication...

To complete authentication:
1. Visit: https://dev-d4be5uc4a3c311t3.us.auth0.com/activate?user_code=XXXX-XXXX
2. Or go to https://dev-d4be5uc4a3c311t3.us.auth0.com/activate and enter code: XXXX-XXXX

Waiting for you to complete authentication in the browser...

✅ Successfully authenticated
Token expires at: 07 Apr 26 23:51 IST
```

### Who Am I

```bash
mump2p whoami
```

```
Authentication Status:
----------------------
Client ID: auth0|<USER_ID>
Expires: 07 Apr 26 23:51 IST
Valid for: 720h0m0s
Is Active:  true

Rate Limits:
------------
Publish Rate:  50000 per hour
Publish Rate:  600 per second
Max Message Size:  10.00 MB
Daily Quota:       20480.00 MB
```

### Logout

```bash
mump2p logout
```

```
✅ Successfully logged out
```

## Subscribe

Subscribe to a topic and stream messages directly from a P2P node. By default, 3 nodes are requested for automatic failover.

```bash
mump2p subscribe --topic test
```

```
Subscribed to 'test' on 34.126.161.115:33211 (Singapore) in 161ms — 2 backup nodes ready
  backup: 34.141.111.130:33211 (Germany)
  backup: 136.110.0.19:33211 (Singapore)
[test] Hello from authenticated CLI!
[test] Second authenticated message
^C
Disconnected — 2 messages in 14s
```

### With single node

```bash
mump2p subscribe --topic test --expose-amount 1
```

```
Subscribed to 'test' on 34.126.161.115:33211 (Singapore) in 175ms
[test] hello world
```

### Failover

If the first node is unreachable, the CLI automatically falls back to the next one:

```
  Node 136.110.0.19:33211 unreachable, falling back...
Subscribed to 'test' on 34.126.161.115:33211 (Singapore) in 312ms — 1 backup node ready
  backup: 34.141.111.130:33211 (Germany)
```

### Persist messages to file

```bash
mump2p subscribe --topic test --persist ./messages.log
```

### Forward to webhook

```bash
mump2p subscribe --topic test --webhook https://hooks.slack.com/services/xxx
mump2p subscribe --topic test --webhook https://discord.com/api/webhooks/xxx --webhook-schema '{"content":"{{.Message}}"}'
```

## Publish

Publish a message to a topic directly to a P2P node.

```bash
mump2p publish --topic test --message "Hello World"
```

```
Published to 34.126.161.115:33211 (Singapore) in 259ms
```

### From file

```bash
mump2p publish --topic test/data --file ./payload.json
```

## Debug Mode

Use `--debug` to see session details, node scores, timing breakdowns, message IDs, and peer paths.

```bash
mump2p subscribe --topic test --debug
```

```
New session 09ddc264-e0bd-42af-b768-feacd512e686 from http://us1-proxy.getoptimum.io:8080 (535ms) | 3 node(s) available
  Trying node 1/3: 34.126.161.115:33211 (Singapore, score: 0.98)...
Subscribed to 'test' on 34.126.161.115:33211 (Singapore) in 188ms — 2 backup nodes ready
  backup: 34.141.111.130:33211 (Germany)
  backup: 136.110.0.19:33211 (Singapore)
Recv:	[1]	receiver_addr:34.126.161.115	[recv_time, size]:[1775977506353069000, 21]	topic:test	hash:65456d67	protocol:gRPC-direct
  from: 12D3KooWSviMsA9yDTxmv1svvGrEPeW6F5DYzSFQihE9YrQW6vtm
  via:  34.126.161.115:33211 (Singapore)
  id:   65456d676c97...
```

```bash
mump2p publish --topic test --message "hello" --debug
```

```
New session 619ceb3b-9440-4654-923f-982bd6dba0be from http://us1-proxy.getoptimum.io:8080 (516ms) | 1 node(s) available
  Trying node 1/1: 34.126.161.115:33211 (Singapore, score: 0.98)...
Session: 516ms | Published: 261ms | Total: 777ms
Published to 34.126.161.115:33211 (Singapore) in 261ms
```

## Health

```bash
mump2p health
```

```
Proxy Health Status:
-------------------
Status:      ok
Memory Used: 51.30%
CPU Used:    73.94%
Disk Used:   9.41%
Country:     United States (US)
```

## List Topics

```bash
mump2p list-topics
```

```
📋 Subscribed Topics for Client: auth0|<USER_ID>
═══════════════════════════════════════════════════════════════
   Total Topics: 7

   1. test/adr2-cli
   2. test/cli-e2e
   3. test
   4. /eth2/c6ecb76c/beacon_block/ssz_snappy
   5. mump2p_aggregated_messages
   6. test/adr2-grpc
   7. test/domain-e2e
═══════════════════════════════════════════════════════════════
```

## Usage Stats

```bash
mump2p usage
```

```
  Publish (hour):     2 / 50000
  Publish (second):   1 / 600
  Data Used:          0.0001 MB / 20480.0000 MB
  Next Reset:         09 Mar 26 23:52 IST (23h58m10s from now)
  Last Publish:       08 Mar 26 23:52 IST
```

## Version

```bash
mump2p version
```

```
Version: v0.0.1-rc8
Commit:  4f76630
```

## Update

```bash
mump2p update
```

## Without Auth (Testing)

All commands support `--disable-auth --client-id <id>` to skip Auth0.

```bash
mump2p subscribe --topic test --disable-auth --client-id my-test-user
mump2p publish --topic test --message "hello" --disable-auth --client-id my-test-user
mump2p list-topics --disable-auth --client-id my-test-user
```

## Output Formats

All read commands support `--output json` or `--output yaml`.

```bash
mump2p whoami --output json
mump2p health --output yaml
mump2p list-topics --output json
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--auth-path` | Custom path for auth file (default: `~/.mump2p/auth.yml`) |
| `--client-id` | Client ID (required with `--disable-auth`) |
| `--debug` | Session detail, node scores, message IDs, peer paths, timing breakdowns |
| `--disable-auth` | Skip Auth0 for testing |
| `--expose-amount N` | Number of nodes to request for failover (subscribe default: `3`, publish default: `1`) |
| `--output` | Output format: `table`, `json`, `yaml` |

## Override Proxy

Any command that talks to the proxy accepts `--service-url`:

```bash
mump2p subscribe --topic test --service-url http://us2-proxy.getoptimum.io:8080
mump2p publish --topic test --message "hi" --service-url http://us3-proxy.getoptimum.io:8080
mump2p health --service-url http://us2-proxy.getoptimum.io:8080
```

Available proxies:
- `http://us1-proxy.getoptimum.io:8080`
- `http://us2-proxy.getoptimum.io:8080`
- `http://us3-proxy.getoptimum.io:8080`
