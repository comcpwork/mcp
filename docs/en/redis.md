# Redis Usage Guide

## MCP Tool

**Tool Name:** `redis_exec`

## DSN Format

```
redis://[:password@]host:port/database
```

### Examples

| Scenario | DSN |
|----------|-----|
| Local without password | `redis://localhost:6379/0` |
| With password | `redis://:mypassword@localhost:6379/0` |
| Remote server | `redis://:pass123@192.168.1.100:6379/1` |
| With username | `redis://user:password@localhost:6379/0` |

## Usage Examples

### String Operations

Ask your AI assistant:

- "Execute Redis command `PING` on `redis://localhost:6379/0`"
- "Execute Redis command `SET user:1001 John` on `redis://localhost:6379/0`"
- "Execute Redis command `GET user:1001` on `redis://localhost:6379/0`"
- "Execute Redis command `DEL session:abc123` on `redis://localhost:6379/0`"

### List Operations

- "Execute Redis command `LPUSH queue task1` on `redis://localhost:6379/0`"
- "Execute Redis command `LRANGE todo_list 0 -1` on `redis://localhost:6379/0`"
- "Execute Redis command `RPOP queue` on `redis://localhost:6379/0`"

### Hash Operations

- "Execute Redis command `HSET user:1001 name John age 30` on `redis://localhost:6379/0`"
- "Execute Redis command `HGETALL user:1001` on `redis://localhost:6379/0`"
- "Execute Redis command `HGET user:1001 name` on `redis://localhost:6379/0`"

### Set Operations

- "Execute Redis command `SADD online_users user123` on `redis://localhost:6379/0`"
- "Execute Redis command `SMEMBERS online_users` on `redis://localhost:6379/0`"
- "Execute Redis command `SISMEMBER online_users user123` on `redis://localhost:6379/0`"

### Key Management

- "Execute Redis command `KEYS user:*` on `redis://localhost:6379/0`"
- "Execute Redis command `TTL session:abc123` on `redis://localhost:6379/0`"
- "Execute Redis command `EXPIRE session:abc123 3600` on `redis://localhost:6379/0`"

## Output Format

### String Values

```
"John"
```

### Integer Values

```
(integer) 1
```

### Nil Values

```
(nil)
```

### Array Values

```
(3 items)
1) "item1"
2) "item2"
3) "item3"
```

### Hash Values

```
(4 fields)
name: John
age: 30
email: john@example.com
status: active
```

## Common Commands

| Command | Description | Example |
|---------|-------------|---------|
| `PING` | Test connection | `PING` |
| `SET` | Set string value | `SET key value` |
| `GET` | Get string value | `GET key` |
| `DEL` | Delete key(s) | `DEL key1 key2` |
| `EXISTS` | Check key exists | `EXISTS key` |
| `KEYS` | Find keys by pattern | `KEYS user:*` |
| `TTL` | Get time-to-live | `TTL key` |
| `EXPIRE` | Set expiration | `EXPIRE key seconds` |
| `HSET` | Set hash field(s) | `HSET hash field value` |
| `HGET` | Get hash field | `HGET hash field` |
| `HGETALL` | Get all hash fields | `HGETALL hash` |
| `LPUSH` | Push to list head | `LPUSH list value` |
| `RPUSH` | Push to list tail | `RPUSH list value` |
| `LRANGE` | Get list range | `LRANGE list 0 -1` |
| `SADD` | Add to set | `SADD set member` |
| `SMEMBERS` | Get all set members | `SMEMBERS set` |

## Tips

1. **Database Selection**: Redis has 16 databases (0-15), specify in DSN path
2. **Key Patterns**: Use `KEYS pattern` carefully in production (blocks server)
3. **TTL Management**: Use `EXPIRE` for session/cache management
4. **Data Types**: Choose appropriate data structure for your use case
