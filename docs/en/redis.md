# Redis Usage Guide

## Getting Started

### Connect to Redis

Tell Claude:
- "Connect to Redis at localhost:6379 with password redis123"
- "Connect to local Redis server"

### Basic Operations

#### Key Operations
- "Set key user:1001 to value John Doe"
- "Get the value of key user:1001"
- "Delete key session:abc123"
- "Check if key exists"

#### List Operations
- "Push item to shopping_cart list"
- "Get all items from todo_list"
- "Remove first item from queue"

#### Hash Operations
- "Set user:1001 name to John and age to 30"
- "Get all fields from user:1001 hash"
- "Update user:1001 email field"

#### Set Operations
- "Add user123 to online_users set"
- "Check if user123 is in online_users"
- "Get all members of admin_users set"

## Advanced Features

### Key Patterns
- "Find all keys starting with session:"
- "Count keys matching user:*"
- "Delete all keys with pattern temp:*"

### Expiration
- "Set key with 1 hour expiration"
- "Check TTL of session key"
- "Remove expiration from key"

### Batch Operations
- "Get multiple keys: key1, key2, key3"
- "Delete keys: temp1, temp2, temp3"
- "Set multiple key-value pairs"

## Tips

1. **Data Types**: Redis supports strings, lists, sets, hashes, and more
2. **Expiration**: Use TTL for session management
3. **Patterns**: Use wildcards carefully in production
4. **Pipeline**: Multiple commands can be executed together