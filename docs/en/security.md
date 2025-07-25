# Security Options

## Overview

MCP tools provide granular security controls to prevent dangerous operations. By default, all operations are allowed, but you can selectively disable specific operations using command-line flags.

## MySQL Security Flags

```bash
# Disable DROP operations
mcp mysql --disable-drop

# Disable multiple operations
mcp mysql --disable-drop --disable-truncate --disable-delete

# Maximum restrictions
mcp mysql --disable-create --disable-drop --disable-alter --disable-truncate --disable-update --disable-delete
```

### Available Flags

| Flag | Description | Blocked Operations |
|------|-------------|-------------------|
| `--disable-create` | Prevent CREATE operations | CREATE DATABASE, CREATE TABLE, CREATE INDEX |
| `--disable-drop` | Prevent DROP operations | DROP DATABASE, DROP TABLE, DROP INDEX |
| `--disable-alter` | Prevent ALTER operations | ALTER TABLE, ALTER DATABASE |
| `--disable-truncate` | Prevent TRUNCATE operations | TRUNCATE TABLE |
| `--disable-update` | Prevent UPDATE operations | UPDATE statements |
| `--disable-delete` | Prevent DELETE operations | DELETE statements |

## Redis Security Flags

```bash
# Disable dangerous commands
mcp redis --disable-delete --disable-update
```

### Blocked Commands

- **--disable-delete**: DEL, UNLINK, FLUSHDB, FLUSHALL
- **--disable-update**: CONFIG, EVAL, EVALSHA, SCRIPT

## Pulsar Security Flags

```bash
# Disable administrative operations
mcp pulsar --disable-create --disable-drop
```

### Blocked Operations

- **--disable-create**: Create tenant/namespace/topic/subscription
- **--disable-drop**: Delete tenant/namespace/topic/subscription
- **--disable-update**: Update configurations

## Best Practices

1. **Development**: Use default settings (all operations allowed)
2. **Staging**: Enable some restrictions for safety
3. **Production**: Maximum restrictions, only allow read operations
4. **CI/CD**: Customize based on pipeline requirements

## Examples

### Read-only MySQL Access
```bash
mcp mysql --disable-create --disable-drop --disable-alter --disable-truncate --disable-update --disable-delete
```

### Safe Redis Access
```bash
mcp redis --disable-delete --disable-update
```

### Limited Pulsar Management
```bash
mcp pulsar --disable-drop --disable-update
```