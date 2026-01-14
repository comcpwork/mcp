# ClickHouse Usage Guide

## MCP Tool

**Tool Name:** `clickhouse_exec`

## DSN Format

```
clickhouse://username:password@host:port/database?options
```

### Examples

| Scenario | DSN |
|----------|-----|
| Local default | `clickhouse://default:@localhost:9000/default` |
| With password | `clickhouse://default:password@localhost:9000/mydb` |
| Remote server | `clickhouse://admin:pass123@192.168.1.100:9000/production` |
| With timeout | `clickhouse://default:@localhost:9000/default?dial_timeout=10s` |
| With read timeout | `clickhouse://default:@localhost:9000/default?read_timeout=20s` |

### Common Options

| Option | Description | Example |
|--------|-------------|---------|
| `dial_timeout` | Connection timeout | `dial_timeout=10s` |
| `read_timeout` | Read timeout | `read_timeout=20s` |
| `write_timeout` | Write timeout | `write_timeout=20s` |
| `compress` | Enable compression | `compress=lz4` |

## Usage Examples

### Query Operations

Ask your AI assistant:

- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/default` and SQL: `SELECT 1`"
- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/default` and SQL: `SHOW DATABASES`"
- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/default` and SQL: `SHOW TABLES`"
- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/mydb` and SQL: `SELECT * FROM events LIMIT 10`"

### Table Operations

- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/default` and SQL: `DESCRIBE TABLE events`"
- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/default` and SQL: `EXISTS TABLE events`"

### Data Modification

- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/mydb` and SQL: `INSERT INTO events (id, name, timestamp) VALUES (1, 'click', now())`"
- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/mydb` and SQL: `ALTER TABLE events ADD COLUMN category String`"

### Analytics Queries

- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/mydb` and SQL: `SELECT toDate(timestamp) as date, count() as cnt FROM events GROUP BY date ORDER BY date`"
- "Execute ClickHouse with DSN `clickhouse://default:@localhost:9000/mydb` and SQL: `SELECT uniq(user_id) FROM events WHERE timestamp > now() - INTERVAL 1 DAY`"

## Supported Operations

### Query Statements
- `SELECT` - Query data
- `SHOW` - Show databases, tables, etc.
- `DESCRIBE` / `DESC` - Describe table structure
- `EXPLAIN` - Explain query plan
- `EXISTS` - Check if table exists

### Modification Statements
- `INSERT` - Insert data
- `ALTER` - Alter table structure
- `CREATE` - Create database/table
- `DROP` - Drop database/table
- `TRUNCATE` - Truncate table

## Output Format

### Query Results

**≤5 columns:** Table format
```
date          cnt
----------    -----
2024-01-01    1000
2024-01-02    1500
2024-01-03    1200
```

**>5 columns:** Key-value format
```
Row 1:
  id: 1
  name: click
  user_id: 12345
  timestamp: 2024-01-01 12:00:00
  ...
```

### Modification Results

```
Query OK, 1 row affected
```

## SSH Connection

Connect to ClickHouse through SSH bastion host using the optional `ssh` parameter.

### SSH URI Formats

| Format | Example | Description |
|--------|---------|-------------|
| Config reference | `ssh://myserver` | Use `~/.ssh/config` entry |
| Password auth | `ssh://user:pass@host:port` | Direct password authentication |
| Key auth | `ssh://user@host?key=/path/to/key` | Private key authentication |

### Examples

**Using SSH config:**
```
DSN: clickhouse://default:@10.0.0.102:9000/mydb
SSH: ssh://myserver
```

**Using SSH key:**
```
DSN: clickhouse://default:password@10.0.0.102:9000/mydb
SSH: ssh://admin@jump.example.com?key=~/.ssh/id_rsa
```

> **Note:** The DSN host:port should be accessible from the SSH server (e.g., internal IP).

## Tips

1. **Port**: ClickHouse native protocol uses port 9000 (not 8123 for HTTP)
2. **Default User**: ClickHouse default user is `default` with empty password
3. **Compression**: Use `compress=lz4` for better performance over network
4. **LIMIT Clause**: Always use `LIMIT` for large tables to avoid memory issues
5. **Time Functions**: ClickHouse has rich time functions like `now()`, `today()`, `toDate()`
