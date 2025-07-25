# MCP 测试最佳实践

## 资源管理问题

### 当前测试的问题

1. **缺少异常保护**
   - 如果测试中途失败，清理代码不会执行
   - 数据库和表会残留在服务器上

2. **测试依赖性**
   - 测试必须按顺序执行
   - 无法单独运行某个测试
   - 一个测试失败会影响后续所有测试

3. **没有使用 pytest 特性**
   - 未使用 fixture 管理资源生命周期
   - 没有 teardown 或 finalizer

## 推荐的测试模式

### 1. 使用 pytest fixtures

```python
@pytest.fixture
async def test_database(self):
    """创建测试数据库，确保清理"""
    db_name = f"test_mcp_{uuid.uuid4().hex[:8]}"
    
    # Setup
    async with self.get_mysql_session(use_root=True) as session:
        await session.call_tool("exec", {
            "sql": f"CREATE DATABASE {db_name}"
        })
    
    yield db_name
    
    # Teardown - 总是会执行
    try:
        async with self.get_mysql_session(use_root=True) as session:
            await session.call_tool("exec", {
                "sql": f"DROP DATABASE IF EXISTS {db_name}"
            })
    except Exception as e:
        print(f"清理失败: {e}")
```

### 2. 使用 try-finally

```python
async def test_with_cleanup(self):
    db_name = f"test_mcp_{uuid.uuid4().hex[:8]}"
    
    try:
        # 创建资源
        await self.create_database(db_name)
        
        # 执行测试
        await self.run_tests(db_name)
        
    finally:
        # 确保清理
        await self.cleanup_database(db_name)
```

### 3. 使用上下文管理器

```python
@asynccontextmanager
async def test_database_context(self):
    db_name = f"test_mcp_{uuid.uuid4().hex[:8]}"
    
    # 创建
    await self.create_database(db_name)
    
    try:
        yield db_name
    finally:
        # 清理
        await self.drop_database(db_name)

# 使用
async def test_something(self):
    async with self.test_database_context() as db_name:
        # 执行测试
        pass
```

## 清理策略

### 1. 定期清理脚本

运行 `cleanup_test_dbs.py` 清理残留的测试数据库：

```bash
python test/cleanup_test_dbs.py
```

### 2. CI/CD 集成

在 CI 管道中添加清理步骤：

```yaml
- name: Run tests
  run: pytest test/

- name: Cleanup test resources
  if: always()  # 总是执行
  run: python test/cleanup_test_dbs.py --auto-confirm
```

### 3. 命名约定

- 所有测试资源使用特定前缀：`test_mcp_`
- 包含时间戳或 UUID 避免冲突
- 便于批量识别和清理

## 测试隔离

### 1. 每个测试独立

```python
@pytest.mark.asyncio
async def test_insert_independent(self, test_table):
    """独立的插入测试"""
    db_name, table_name = test_table
    # 这个测试不依赖其他测试
    ...
```

### 2. 使用事务回滚

```python
async def test_with_rollback(self):
    async with self.get_mysql_session() as session:
        # 开始事务
        await session.call_tool("exec", {"sql": "START TRANSACTION"})
        
        try:
            # 执行测试操作
            await self.perform_test_operations()
            
        finally:
            # 回滚所有更改
            await session.call_tool("exec", {"sql": "ROLLBACK"})
```

## 监控和报告

### 1. 资源使用追踪

```python
class ResourceTracker:
    def __init__(self):
        self.created_resources = []
    
    def track(self, resource_type, resource_name):
        self.created_resources.append({
            'type': resource_type,
            'name': resource_name,
            'created_at': datetime.now()
        })
    
    def cleanup_all(self):
        for resource in self.created_resources:
            self.cleanup_resource(resource)
```

### 2. 测试报告

在测试结束时生成资源使用报告：

```python
def pytest_sessionfinish(session, exitstatus):
    """在所有测试结束后运行"""
    print("\n=== 资源使用报告 ===")
    print(f"创建的数据库: {created_databases}")
    print(f"清理的数据库: {cleaned_databases}")
    print(f"残留的资源: {remaining_resources}")
```

## 安全考虑

1. **使用专用测试账号**
   - 限制权限，只允许操作测试数据库
   - 避免误删生产数据

2. **环境隔离**
   - 开发、测试、生产环境分离
   - 使用不同的配置文件

3. **数据保护**
   - 不要在测试中使用真实数据
   - 敏感信息使用环境变量

## 推荐工具

1. **pytest-asyncio**: 异步测试支持
2. **pytest-timeout**: 防止测试卡住
3. **pytest-xdist**: 并行执行测试
4. **pytest-repeat**: 重复运行测试检查稳定性

## 总结

好的测试应该是：
- ✅ **自清理的**: 无论成功或失败都会清理资源
- ✅ **独立的**: 不依赖其他测试或执行顺序
- ✅ **可重复的**: 多次运行结果一致
- ✅ **隔离的**: 不影响其他测试或系统
- ✅ **快速的**: 合理使用资源，及时释放