# Pulsar Usage Guide

## Getting Started

### Connect to Pulsar

Tell Claude:
- "Connect to Pulsar at http://localhost:8080"
- "Connect to Pulsar admin API"

### Basic Operations

#### Tenant Management
- "List all tenants"
- "Create tenant my-tenant"
- "Delete tenant test-tenant"

#### Namespace Management
- "List namespaces in public tenant"
- "Create namespace public/my-namespace"
- "Delete namespace public/test"

#### Topic Management
- "Create topic persistent://public/default/my-topic"
- "List all topics in namespace"
- "Delete topic my-topic"
- "Get topic statistics"

#### Subscription Management
- "Create subscription my-sub on topic my-topic"
- "List subscriptions for topic"
- "Delete subscription my-sub"

## Advanced Features

### Topic Information
- "Get detailed info about topic my-topic"
- "Show topic partitions"
- "Check topic backlog"

### Broker Management
- "List all active brokers"
- "Get broker load information"
- "Check broker health"

### Batch Operations
- "Get info for multiple topics"
- "Create multiple topics at once"

## Tips

1. **Persistence**: Use `persistent://` for durable topics
2. **Naming**: Follow the pattern `persistent://tenant/namespace/topic`
3. **Partitions**: Specify partitions for high throughput
4. **Subscriptions**: Multiple subscriptions enable different consumption patterns