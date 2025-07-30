# OptimumP2P CLI - Advanced Usage Guide

This guide covers advanced OptimumP2P CLI features and usage patterns. Complete the [basic guide](./guide.md) first before proceeding.

## Prerequisites

- Completed basic guide setup and authentication
- Built CLI


---

## Advanced Subscription Patterns

### Multi-Topic Concurrent Subscriptions

Run multiple subscriptions simultaneously across different terminals:

**Terminal 1 - Basic subscription:**
```bash
./dist/mump2p-mac subscribe --topic=events --service-url="http://35.221.118.95:8080"
```

**Terminal 2 - Subscription with persistence:**
```bash
mkdir -p logs/data-stream
./dist/mump2p-mac subscribe --topic=data-stream --persist=logs/data-stream/ --service-url="http://35.221.118.95:8080"
```

**Terminal 3 - High-threshold subscription:**
```bash
./dist/mump2p-mac subscribe --topic=critical-events --threshold=0.9 --service-url="http://35.221.118.95:8080"
```

### Webhook Integration

Forward messages to HTTP endpoints for integration with external systems:

```bash
# Basic webhook forwarding (use webhook.site for testing)
./dist/mump2p-mac subscribe --topic=webhook-test \
  --webhook=https://webhook.site/unique-url \
  --service-url="http://35.221.118.95:8080"

# Advanced webhook configuration
./dist/mump2p-mac subscribe --topic=high-volume \
  --webhook=https://your-api.example.com/webhook \
  --webhook-queue-size=500 \
  --webhook-timeout=10 \
  --service-url="http://35.221.118.95:8080"
```

### Combined Persistence and Webhooks

```bash
./dist/mump2p-mac subscribe --topic=audit-trail \
  --persist=logs/audit/messages.log \
  --webhook=https://your-monitoring.example.com/events \
  --webhook-queue-size=200 \
  --webhook-timeout=5 \
  --service-url="http://35.221.118.95:8080"
```

---

## Advanced Publishing Patterns

### File-Based Publishing

Create and publish various data formats:

```bash
# JSON configuration
cat > config-update.json << 'EOF'
{
  "version": "2.1.0",
  "features": {
    "enhanced_routing": true,
    "compression": "lz4",
    "encryption": "aes256"
  },
  "timestamp": "2025-07-30T12:00:00Z"
}
EOF

./dist/mump2p-mac publish --topic=config-updates --file=config-update.json --service-url="http://35.221.118.95:8080"
```

### Structured Data Publishing

```bash
# System metrics
./dist/mump2p-mac publish --topic=metrics --message='{
  "system": {
    "cpu_usage": 45.2,
    "memory_usage": 78.5,
    "disk_usage": 23.1,
    "network_io": {
      "tx": 1024000,
      "rx": 2048000
    }
  },
  "timestamp": "2025-07-30T12:00:00Z"
}' --service-url="http://35.221.118.95:8080"

# Transaction data
./dist/mump2p-mac publish --topic=transactions --message='{
  "transaction_id": "tx_1234567890",
  "from": "0xabc123...",
  "to": "0xdef456...",
  "amount": "1000000000000000000",
  "gas_used": 21000,
  "block_number": 18500000
}' --service-url="http://35.221.118.95:8080"
```

### Batch Publishing

```bash
# Publish multiple messages with delays
for i in {1..10}; do
  ./dist/mump2p-mac publish --topic=batch-test --message="Batch message $i - $(date)" --service-url="http://35.221.118.95:8080"
  sleep 0.5
done
```

---

## Real-Time Communication Patterns

### Event-Driven Architecture

**Event Publisher:**
```bash
# Publish system events
./dist/mump2p-mac publish --topic=system-events --message='{"event": "service_started", "service": "api-gateway", "timestamp": "2025-07-30T12:00:00Z"}' --service-url="http://35.221.118.95:8080"

./dist/mump2p-mac publish --topic=system-events --message='{"event": "high_cpu_usage", "cpu": 89.5, "threshold": 80, "timestamp": "2025-07-30T12:01:00Z"}' --service-url="http://35.221.118.95:8080"
```

**Event Subscriber:**
```bash
./dist/mump2p-mac subscribe --topic=system-events --persist=logs/events/ --service-url="http://35.221.118.95:8080"
```

### Data Pipeline Simulation

**Data Producer:**
```bash
# Simulate sensor data pipeline
for i in {1..20}; do
  ./dist/mump2p-mac publish --topic=sensor-data --message="{
    \"sensor_id\": \"temp_$(($i % 5 + 1))\",
    \"value\": $((20 + $RANDOM % 20)),
    \"unit\": \"celsius\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
  }" --service-url="http://35.221.118.95:8080"
  sleep 1
done
```

**Data Consumer:**
```bash
./dist/mump2p-mac subscribe --topic=sensor-data \
  --persist=logs/sensors/ \
  --service-url="http://35.221.118.95:8080"
```

---

## Multi-Gateway Operations

### Cross-Region Message Distribution

Test message routing across different geographic regions:

```bash
# Publish from Tokyo
./dist/mump2p-mac publish --topic=global-sync --message="Message from Tokyo region" --service-url="http://35.221.118.95:8080"

# Publish from Singapore  
./dist/mump2p-mac publish --topic=global-sync --message="Message from Singapore region" --service-url="http://34.142.205.26:8080"

# Subscribe from primary gateway
./dist/mump2p-mac subscribe --topic=global-sync --service-url="http://34.146.222.111:8080"
```

### Gateway Performance Testing

```bash
# Test different gateways with the same topic
./dist/mump2p-mac publish --topic=perf-test --message="Performance test - Gateway 1" --service-url="http://34.146.222.111:8080"
./dist/mump2p-mac publish --topic=perf-test --message="Performance test - Gateway 2" --service-url="http://35.221.118.95:8080"
./dist/mump2p-mac publish --topic=perf-test --message="Performance test - Gateway 3" --service-url="http://34.142.205.26:8080"
```

---

### Large File Handling

```bash
# Create larger test files
echo "Creating test content..." > large-content.txt
for i in {1..500}; do 
  echo "Line $i: This is test content for large file publishing to test size limits and performance" >> large-content.txt
done

./dist/mump2p-mac publish --topic=large-files --file=large-content.txt --service-url="http://35.221.118.95:8080"
```

---

## Multi-Terminal Demo Scenarios

### Chat Application Simulation

**Terminal 1 - Chat Subscriber:**
```bash
./dist/mump2p-mac subscribe --topic=chat-room --persist=logs/chat/ --service-url="http://35.221.118.95:8080"
```

**Terminal 2 - User 1:**
```bash
./dist/mump2p-mac publish --topic=chat-room --message="[Alice] Hello everyone!" --service-url="http://35.221.118.95:8080"
./dist/mump2p-mac publish --topic=chat-room --message="[Alice] How is everyone doing?" --service-url="http://35.221.118.95:8080"
```

**Terminal 3 - User 2:**
```bash
./dist/mump2p-mac publish --topic=chat-room --message="[Bob] Hey Alice! Great to see you here!" --service-url="http://35.221.118.95:8080"
./dist/mump2p-mac publish --topic=chat-room --message="[Bob] OptimumP2P is working perfectly!" --service-url="http://35.221.118.95:8080"
```

### IoT Data Collection

**Terminal 1 - Data Collector:**
```bash
./dist/mump2p-mac subscribe --topic=iot-data --persist=logs/iot-data/ --service-url="http://35.221.118.95:8080"
```

**Terminal 2 - Temperature Sensors:**
```bash
for i in {1..10}; do
  temp=$((20 + $RANDOM % 15))
  ./dist/mump2p-mac publish --topic=iot-data --message="{\"sensor\": \"temp_01\", \"value\": $temp, \"unit\": \"celsius\", \"time\": \"$(date)\"}" --service-url="http://35.221.118.95:8080"
  sleep 2
done
```

**Terminal 3 - Humidity Sensors:**
```bash
for i in {1..10}; do
  humidity=$((40 + $RANDOM % 40))
  ./dist/mump2p-mac publish --topic=iot-data --message="{\"sensor\": \"humidity_01\", \"value\": $humidity, \"unit\": \"percent\", \"time\": \"$(date)\"}" --service-url="http://35.221.118.95:8080"
  sleep 3
done
```

---

## Usage Monitoring

```bash
# Check current usage and limits
./dist/mump2p-mac usage

# Check authentication status
./dist/mump2p-mac whoami

# Refresh token if needed
./dist/mump2p-mac refresh
```

---

## Cleanup

```bash
# Clean up test files
rm -rf logs/ *.txt *.json
```

---


