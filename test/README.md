# MCP 项目测试框架

## 概述

本测试框架基于 Python MCP SDK 设计，用于对 MCP 项目的三个 provider（MySQL、Redis、Pulsar）进行全面的集成测试和功能验证。

## 项目结构

```
test/
├── README.md                   # 本文件
├── requirements.txt            # 测试依赖包
├── conftest.py                 # pytest 配置和公共 fixtures
├── client/                     # 通用 MCP 客户端工具
│   ├── __init__.py
│   ├── base_client.py         # 基础客户端类
│   └── test_helpers.py        # 测试辅助函数
├── fixtures/                   # 测试数据和配置
│   ├── config/                # 配置文件模板
│   │   ├── mysql_config.yaml
│   │   ├── redis_config.yaml
│   │   └── pulsar_config.yaml
│   └── data/                  # 测试数据
│       ├── mysql_test_data.sql
│       └── sample_data.json
├── integration/                # 集成测试
│   ├── test_mysql_provider.py # MySQL provider 测试
│   ├── test_redis_provider.py # Redis provider 测试
│   └── test_pulsar_provider.py# Pulsar provider 测试
├── unit/                       # 单元测试
│   ├── test_common_package.py # common 包测试
│   └── test_error_handling.py # 错误处理测试
└── utils/                      # 测试工具
    ├── docker_manager.py      # Docker 容器管理
    ├── server_launcher.py     # MCP 服务器启动器
    └── data_generator.py      # 测试数据生成器
```

## 测试覆盖范围

### MySQL Provider (10 个命令)
- 连接管理: `connect_mysql`, `current_mysql`, `history_mysql`, `switch_mysql`
- 数据操作: `exec` (SELECT/INSERT/UPDATE/DELETE)
- Schema 操作: `show_tables`, `describe_table`, `describe_tables`
- 配置管理: `update_config`, `get_config_details`

### Redis Provider (7 个命令)
- 连接管理: `connect_redis`, `current_redis`, `history_redis`, `switch_redis`
- 命令执行: `exec` (单命令和管道模式)
- 配置管理: `update_config`, `get_config_details`

### Pulsar Provider (26 个命令)
- 连接管理: `connect_pulsar`, `current_pulsar`, `history_pulsar`, `switch_pulsar`
- 租户管理: `list_tenants`, `create_tenant`, `delete_tenant`, `get_tenant_info`
- 命名空间管理: `list_namespaces`, `create_namespace`, `delete_namespace`, `get_namespace_info`
- 主题管理: `list_topics`, `create_topic`, `delete_topic`, `get_topic_stats`, `get_topic_info`
- 订阅管理: `list_subscriptions`, `create_subscription`, `delete_subscription`
- Broker 管理: `list_brokers`, `get_broker_info`, `broker_healthcheck`
- 配置管理: `update_config`, `get_config_details`

## 技术栈

- **测试框架**: pytest + pytest-asyncio
- **MCP 客户端**: 官方 MCP Python SDK
- **数据库环境**: Docker 容器 (MySQL 8.0, Redis 7.0, Apache Pulsar 3.0)
- **并行测试**: pytest-xdist
- **容器管理**: testcontainers

## 快速开始

### 1. 安装依赖

```bash
cd test
pip install -r requirements.txt
```

### 2. 启动测试环境

```bash
# 启动 Docker 容器
docker-compose up -d

# 运行所有测试
pytest

# 运行特定 provider 测试
pytest integration/test_mysql_provider.py
pytest integration/test_redis_provider.py
pytest integration/test_pulsar_provider.py
```

### 3. 并行测试

```bash
# 使用多进程并行测试
pytest -n auto

# 指定进程数
pytest -n 4
```

## 测试原则

1. **真实环境**: 使用真实的数据库服务进行测试
2. **数据隔离**: 每个测试用例使用独立的数据库/命名空间
3. **自动清理**: 测试完成后自动清理测试数据
4. **全面覆盖**: 涵盖功能测试、错误处理、边界条件
5. **高效执行**: 支持并行测试，快速反馈

## 开发指南

### 添加新测试

1. 在相应的 `integration/test_*_provider.py` 中添加测试方法
2. 使用 `@pytest.mark.asyncio` 装饰异步测试方法
3. 利用 `conftest.py` 中的 fixtures 获取测试资源
4. 遵循命名约定: `test_<功能名>_<场景>`

### 测试数据管理

1. 静态测试数据放在 `fixtures/data/` 目录下
2. 动态测试数据使用 `utils/data_generator.py` 生成
3. 测试配置文件放在 `fixtures/config/` 目录下

### Docker 环境管理

1. 使用 `utils/docker_manager.py` 统一管理容器
2. 支持自动启动、停止、重置容器
3. 提供健康检查和状态监控

## 性能基准

- MySQL 连接时间: < 1s
- Redis 命令执行: < 100ms
- Pulsar 操作响应: < 2s
- 完整测试套件运行: < 5min (并行模式)

## 故障排除

### 常见问题

1. **Docker 容器启动失败**: 检查端口占用和 Docker 服务状态
2. **连接超时**: 增加超时时间或检查网络配置
3. **权限错误**: 确保测试用户有足够的数据库权限
4. **并发冲突**: 检查测试数据隔离是否正确

### 调试模式

```bash
# 启用详细日志
pytest -v -s

# 只运行失败的测试
pytest --lf

# 进入调试器
pytest --pdb
```

## 贡献指南

1. 遵循 [MCP 项目开发要求](../CLAUDE.md)
2. 所有测试必须通过 CI/CD 检查
3. 新增测试需要包含文档说明
4. 性能敏感的测试需要基准对比