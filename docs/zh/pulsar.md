# Pulsar 使用指南

## 开始使用

### 连接到 Pulsar

告诉 Claude：
- "连接到 http://localhost:8080 的 Pulsar"
- "连接到 Pulsar 管理 API"

### 基本操作

#### 租户管理
- "列出所有租户"
- "创建租户 my-tenant"
- "删除租户 test-tenant"

#### 命名空间管理
- "列出 public 租户的命名空间"
- "创建命名空间 public/my-namespace"
- "删除命名空间 public/test"

#### 主题管理
- "创建主题 persistent://public/default/my-topic"
- "列出命名空间中的所有主题"
- "删除主题 my-topic"
- "获取主题统计信息"

#### 订阅管理
- "在主题 my-topic 上创建订阅 my-sub"
- "列出主题的订阅"
- "删除订阅 my-sub"

## 高级功能

### 主题信息
- "获取主题 my-topic 的详细信息"
- "显示主题分区"
- "检查主题积压"

### Broker 管理
- "列出所有活跃的 broker"
- "获取 broker 负载信息"
- "检查 broker 健康状态"

### 批量操作
- "获取多个主题的信息"
- "一次创建多个主题"

## 使用技巧

1. **持久性**：使用 `persistent://` 创建持久主题
2. **命名规范**：遵循 `persistent://租户/命名空间/主题` 格式
3. **分区**：为高吞吐量指定分区数
4. **订阅**：多个订阅支持不同的消费模式