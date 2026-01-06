# MySQL Usage Guide

## MCP Tool

**Tool Name:** `mysql_exec`

## DSN Format

```
username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=true
```

### Examples

| Scenario | DSN |
|----------|-----|
| Local database | `root:password@tcp(localhost:3306)/mydb` |
| Remote server | `admin:pass123@tcp(192.168.1.100:3306)/production` |
| With charset | `root:pass@tcp(localhost:3306)/mydb?charset=utf8mb4` |
| Parse time | `root:pass@tcp(localhost:3306)/mydb?parseTime=true` |

## Usage Examples

### Query Operations

Ask your AI assistant:

- "Execute MySQL with DSN `root:password@tcp(localhost:3306)/mydb` and SQL: `SHOW TABLES`"
- "Execute MySQL with DSN `root:password@tcp(localhost:3306)/mydb` and SQL: `SELECT * FROM users LIMIT 10`"
- "Execute MySQL with DSN `root:password@tcp(localhost:3306)/mydb` and SQL: `DESCRIBE users`"

### Modification Operations

- "Execute MySQL with DSN `root:password@tcp(localhost:3306)/mydb` and SQL: `INSERT INTO users (name, email) VALUES ('John', 'john@example.com')`"
- "Execute MySQL with DSN `root:password@tcp(localhost:3306)/mydb` and SQL: `UPDATE users SET status = 'active' WHERE id = 123`"
- "Execute MySQL with DSN `root:password@tcp(localhost:3306)/mydb` and SQL: `DELETE FROM logs WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY)`"

## Supported Operations

### Query Statements
- `SELECT` - Query data
- `SHOW` - Show database objects
- `DESCRIBE` / `DESC` - Describe table structure
- `EXPLAIN` - Explain query plan

### Modification Statements
- `INSERT` - Insert data
- `UPDATE` - Update data
- `DELETE` - Delete data
- `CREATE` - Create database/table
- `DROP` - Drop database/table
- `ALTER` - Alter table structure
- `TRUNCATE` - Truncate table

## Output Format

### Query Results

**≤5 columns:** Table format
```
ID    Name    Email
----  ------  -----------------
1     John    john@example.com
2     Jane    jane@example.com
```

**>5 columns:** Key-value format
```
Row 1:
  id: 1
  name: John
  email: john@example.com
  ...
```

### Modification Results

```
Query OK, 1 row affected
Last Insert ID: 42
```

## Tips

1. **DSN Security**: Never expose DSN with sensitive credentials in logs or shared environments
2. **LIMIT Clause**: Use `LIMIT` to control result set size for large tables
3. **Natural Language**: You can describe what you want, and the AI will construct the appropriate SQL
