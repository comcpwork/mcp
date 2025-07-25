# MySQL Usage Guide

## Getting Started

### Connect to MySQL

Tell Claude:
- "Connect to MySQL at 192.168.1.100:3306 with username root and password mypass"
- "Connect to local MySQL database"

### Basic Operations

#### View Tables
- "Show all tables"
- "List tables in the database"

#### Query Data
- "Query the users table"
- "Show first 10 records from orders"
- "Find users registered today"

#### Modify Data
- "Insert a new user John Doe with email john@example.com"
- "Update user status to active where id is 123"
- "Delete records older than 30 days from logs table"

## Advanced Features

### Table Information
- "Describe the structure of users table"
- "Show indexes on products table"
- "Check table size and row count"

### Database Management
- "Create a new database called test_db"
- "Show all databases"
- "Switch to production database"

### Complex Queries
- "Show monthly sales summary for last 6 months"
- "Find duplicate email addresses in users table"
- "Join orders with customers and show recent purchases"

## Security Options

When starting the MCP server, you can add security flags:

```bash
mcp mysql --disable-drop --disable-truncate
```

Available flags:
- `--disable-create` - Prevent CREATE operations
- `--disable-drop` - Prevent DROP operations
- `--disable-alter` - Prevent ALTER operations
- `--disable-truncate` - Prevent TRUNCATE operations
- `--disable-update` - Prevent UPDATE operations
- `--disable-delete` - Prevent DELETE operations

## Tips

1. **Natural Language**: Just describe what you want in plain language
2. **Context**: Claude remembers your database context during the conversation
3. **Safety**: Dangerous operations will ask for confirmation
4. **History**: Use "show connection history" to see previous connections