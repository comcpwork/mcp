package mysql

// DefaultConfig MySQL服务器默认配置
const DefaultConfig = `# MySQL MCP 服务器配置
server:
  name: "MySQL MCP Server"
  version: "1.0.0"

# 多数据库连接配置
databases:
  # 默认数据库配置
  default:
    host: "localhost"
    port: 3306
    user: "root"
    password: ""
    database: ""
    charset: "utf8mb4"
    max_connections: 10
    max_idle_connections: 5
    connection_timeout: "30s"

# 当前激活的数据库
active_database: "default"

# 工具配置
tools:
  query:
    timeout: "30s"
    max_rows: 1000
`
