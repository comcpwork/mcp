package pulsar

// DefaultConfig Pulsar的默认配置
const DefaultConfig = `# Pulsar MCP服务器配置

# 当前激活的Pulsar实例
active_database: default

# Pulsar实例配置 - 每个配置对应一个租户下的命名空间
databases:
  default:
    # Pulsar Admin API地址
    admin_url: "http://localhost:8080"
    # 租户名称
    tenant: "public"
    # 命名空间名称
    namespace: "default"
    # 认证用户名（可选）
    username: ""
    # 认证密码（可选）
    password: ""
    # 请求超时时间
    timeout: "30s"

# 工具配置
tools:
  # 默认配置，确保所有工具都有基本设置
  list_pulsar:
    enabled: true
  add_pulsar:
    enabled: true
  set_active_pulsar:
    enabled: true
  remove_pulsar:
    enabled: true
  list_tenants:
    enabled: true
  create_tenant:
    enabled: true
  delete_tenant:
    enabled: true
  list_namespaces:
    enabled: true
  create_namespace:
    enabled: true
  delete_namespace:
    enabled: true
  list_topics:
    enabled: true
  create_topic:
    enabled: true
  delete_topic:
    enabled: true
  get_topic_stats:
    enabled: true
  list_subscriptions:
    enabled: true
  create_subscription:
    enabled: true
  delete_subscription:
    enabled: true
`