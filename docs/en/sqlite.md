# SQLite Usage Guide

## MCP Tool

**Tool Name:** `sqlite_exec`

## DSN Format

```
/path/to/database.db
:memory:
```

### Examples

| Scenario | DSN |
|----------|-----|
| In-memory database | `:memory:` |
| Absolute path | `/Users/<username>/data/mydb.db` |
| Relative path | `./data/test.db` |
| Home directory | `~/databases/app.db` |

**Note:** Replace `<username>` with your actual username.

## Usage Examples

### In-Memory Database

Ask your AI assistant:

- "Execute SQLite with DSN `:memory:` and SQL: `SELECT 1`"
- "Execute SQLite with DSN `:memory:` and SQL: `CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`"
- "Execute SQLite with DSN `:memory:` and SQL: `INSERT INTO users (name) VALUES ('John')`"

**Note:** In-memory databases are temporary and data is lost when the connection closes.

### File-Based Database

- "Execute SQLite with DSN `/Users/john/data/mydb.db` and SQL: `SHOW TABLES`"
- "Execute SQLite with DSN `./test.db` and SQL: `SELECT * FROM users`"
- "Execute SQLite with DSN `~/app.db` and SQL: `DESCRIBE users`"

### Table Operations

- "Execute SQLite with DSN `./mydb.db` and SQL: `CREATE TABLE products (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, price REAL)`"
- "Execute SQLite with DSN `./mydb.db` and SQL: `DROP TABLE IF EXISTS temp_data`"

### Data Operations

- "Execute SQLite with DSN `./mydb.db` and SQL: `INSERT INTO products (name, price) VALUES ('Widget', 9.99)`"
- "Execute SQLite with DSN `./mydb.db` and SQL: `UPDATE products SET price = 12.99 WHERE name = 'Widget'`"
- "Execute SQLite with DSN `./mydb.db` and SQL: `DELETE FROM products WHERE id = 1`"

## Supported Operations

### Query Statements
- `SELECT` - Query data
- `SHOW TABLES` - List all tables (SQLite-specific implementation)
- `DESCRIBE` / `DESC` - Describe table structure
- `EXPLAIN` - Explain query plan

### Modification Statements
- `INSERT` - Insert data
- `UPDATE` - Update data
- `DELETE` - Delete data
- `CREATE` - Create table
- `DROP` - Drop table
- `ALTER` - Alter table structure

## Output Format

### Query Results

**≤5 columns:** Table format
```
id    name     price
----  -------  ------
1     Widget   9.99
2     Gadget   19.99
```

**>5 columns:** Key-value format
```
Row 1:
  id: 1
  name: Widget
  description: A useful widget
  price: 9.99
  stock: 100
  created_at: 2024-01-01
```

### Modification Results

```
Query OK, 1 row affected
Last Insert ID: 42
```

## SQLite Data Types

| Type | Description |
|------|-------------|
| `INTEGER` | Signed integer |
| `REAL` | Floating point |
| `TEXT` | Text string |
| `BLOB` | Binary data |
| `NULL` | Null value |

## Common Patterns

### Auto-increment Primary Key

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);
```

### Timestamps

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY,
    message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Foreign Keys

```sql
CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## Tips

1. **In-Memory vs File**: Use `:memory:` for testing, file path for persistence
2. **File Permissions**: Ensure the directory exists and is writable
3. **AUTOINCREMENT**: Use `INTEGER PRIMARY KEY AUTOINCREMENT` for auto-increment IDs
4. **Concurrent Access**: SQLite supports limited concurrent writes
5. **Backup**: SQLite databases are single files - easy to backup/copy
