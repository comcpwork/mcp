package redis

// DefaultConfig Redis服务器默认配置
const DefaultConfig = `# Redis MCP Server Configuration

# 激活的Redis实例名称
active_redis: default

# Redis实例配置
redis:
  default:
    # Redis服务器地址
    host: localhost
    # Redis服务器端口
    port: 6379
    # Redis密码
    password: "g7z4n1v0b6"
    # 连接数据库编号（0-15）
    database: 0
    # 连接超时时间
    connection_timeout: "5s"
    # 读写超时时间
    read_timeout: "3s"
    write_timeout: "3s"
    # 最大连接数
    max_connections: 10
    # 最大空闲连接数  
    max_idle_connections: 5
    # 连接最大空闲时间
    max_idle_time: "300s"

# 工具配置
tools:
  # 通用配置
  scan:
    # SCAN命令默认每次返回的key数量
    count: 100
  keys:
    # KEYS命令最大返回数量（安全限制）
    max_keys: 1000
  # List操作配置
  list:
    # LRANGE等命令默认最大返回数量
    max_range: 1000
  # Set操作配置  
  set:
    # SMEMBERS等命令默认最大返回数量
    max_members: 1000
  # Hash操作配置
  hash:
    # HGETALL等命令默认最大返回数量
    max_fields: 1000
  # Sorted Set操作配置
  zset:
    # ZRANGE等命令默认最大返回数量
    max_range: 1000

# 日志配置
logging:
  level: info
  output: file
  file_path: ~/.co-mcp/logs/redis.log
`