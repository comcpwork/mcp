"""
MySQL Provider 集成测试（改进版）

使用 pytest fixtures 确保资源正确清理
"""

import asyncio
import yaml
import pytest
import uuid
from contextlib import asynccontextmanager
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


class TestMySQLImproved:
    @classmethod
    def setup_class(cls):
        """加载配置"""
        with open('config.yaml', 'r') as f:
            cls.config = yaml.safe_load(f)
    
    @asynccontextmanager
    async def get_mysql_session(self, use_root=False, database=None):
        """获取 MySQL MCP 会话的上下文管理器"""
        server_params = StdioServerParameters(
            command=self.config["mcp_binary"],
            args=["mysql"],
        )
        
        async with stdio_client(server_params) as (read, write):
            async with ClientSession(read, write) as session:
                await session.initialize()
                
                # 连接到 MySQL
                if use_root:
                    connect_args = {
                        "host": self.config["mysql"]["host"],
                        "port": self.config["mysql"]["port"],
                        "user": self.config["mysql"]["root_user"],
                        "password": self.config["mysql"]["root_password"],
                    }
                else:
                    connect_args = {
                        "host": self.config["mysql"]["host"],
                        "port": self.config["mysql"]["port"],
                        "user": self.config["mysql"]["user"],
                        "password": self.config["mysql"]["password"],
                    }
                
                if database:
                    connect_args["database"] = database
                    
                result = await session.call_tool("connect_mysql", connect_args)
                
                # 验证连接成功
                content = result.content[0].text
                assert any(success_indicator in content for success_indicator in 
                          ["✅", "成功", "Connected", "successfully"]), f"连接失败: {content}"
                
                yield session

    @pytest.fixture
    async def test_database(self):
        """创建测试数据库的 fixture，确保清理"""
        db_name = f"test_mcp_{uuid.uuid4().hex[:8]}"
        
        # Setup: 创建数据库
        async with self.get_mysql_session(use_root=True) as session:
            result = await session.call_tool("exec", {
                "sql": f"CREATE DATABASE {db_name} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
            })
            content = result.content[0].text
            assert any(success in content for success in ["Query OK", "执行成功", "success"])
            print(f"✅ 创建测试数据库: {db_name}")
        
        # 返回数据库名供测试使用
        yield db_name
        
        # Teardown: 无论测试是否成功都清理数据库
        try:
            async with self.get_mysql_session(use_root=True) as session:
                # 先尝试删除可能存在的表
                await session.call_tool("exec", {"sql": f"DROP DATABASE IF EXISTS {db_name}"})
                print(f"✅ 清理测试数据库: {db_name}")
        except Exception as e:
            print(f"⚠️ 清理数据库时出错: {e}")

    @pytest.fixture
    async def test_table(self, test_database):
        """创建测试表的 fixture"""
        table_name = "test_users"
        
        # Setup: 创建表
        async with self.get_mysql_session(use_root=True, database=test_database) as session:
            create_table_sql = f"""
            CREATE TABLE {table_name} (
                id INT AUTO_INCREMENT PRIMARY KEY,
                name VARCHAR(100) NOT NULL COMMENT '用户姓名',
                email VARCHAR(255) UNIQUE NOT NULL COMMENT '邮箱地址',
                age INT DEFAULT 0 COMMENT '年龄',
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'
            ) ENGINE=InnoDB COMMENT='测试用户表'
            """
            
            result = await session.call_tool("exec", {"sql": create_table_sql})
            content = result.content[0].text
            assert any(success in content for success in ["Query OK", "执行成功", "success"])
            print(f"✅ 创建测试表: {table_name}")
        
        yield test_database, table_name
        
        # 表会随数据库一起删除，无需单独清理

    @pytest.mark.asyncio
    async def test_connection_info(self):
        """测试连接信息显示"""
        async with self.get_mysql_session(use_root=True) as session:
            result = await session.call_tool("current_mysql", {})
            content = result.content[0].text
            assert "47.119.182.61" in content
            assert "15640" in content
            print(f"当前连接信息: {content}")

    @pytest.mark.asyncio
    async def test_crud_operations(self, test_table):
        """测试完整的 CRUD 操作"""
        db_name, table_name = test_table
        
        async with self.get_mysql_session(use_root=True, database=db_name) as session:
            # 1. INSERT - 插入数据
            result = await session.call_tool("exec", {
                "sql": f"INSERT INTO {table_name} (name, email, age) VALUES ('张三', 'zhangsan@test.com', 25)"
            })
            assert any(success in result.content[0].text for success in ["1 row", "影响行数: 1", "affected"])
            
            # 批量插入
            batch_insert_sql = f"""
            INSERT INTO {table_name} (name, email, age) VALUES 
            ('李四', 'lisi@test.com', 30),
            ('王五', 'wangwu@test.com', 28),
            ('赵六', 'zhaoliu@test.com', 35)
            """
            result = await session.call_tool("exec", {"sql": batch_insert_sql})
            assert any(success in result.content[0].text for success in ["3 row", "影响行数: 3", "affected"])
            print("✅ 插入测试通过")
            
            # 2. SELECT - 查询数据
            result = await session.call_tool("exec", {
                "sql": f"SELECT * FROM {table_name} ORDER BY id"
            })
            content = result.content[0].text
            assert "张三" in content and "zhangsan@test.com" in content
            
            # 条件查询
            result = await session.call_tool("exec", {
                "sql": f"SELECT name, age FROM {table_name} WHERE age > 30"
            })
            content = result.content[0].text
            assert "赵六" in content and "35" in content
            print("✅ 查询测试通过")
            
            # 3. UPDATE - 更新数据
            result = await session.call_tool("exec", {
                "sql": f"UPDATE {table_name} SET age = 26 WHERE name = '张三'"
            })
            assert any(success in result.content[0].text for success in ["1 row", "影响行数: 1", "affected"])
            
            # 验证更新
            result = await session.call_tool("exec", {
                "sql": f"SELECT age FROM {table_name} WHERE name = '张三'"
            })
            assert "26" in result.content[0].text
            print("✅ 更新测试通过")
            
            # 4. DELETE - 删除数据
            result = await session.call_tool("exec", {
                "sql": f"DELETE FROM {table_name} WHERE name = '王五'"
            })
            assert any(success in result.content[0].text for success in ["1 row", "影响行数: 1", "affected"])
            
            # 验证删除
            result = await session.call_tool("exec", {
                "sql": f"SELECT COUNT(*) as total FROM {table_name}"
            })
            assert "3" in result.content[0].text
            print("✅ 删除测试通过")

    @pytest.mark.asyncio
    async def test_table_operations(self, test_database):
        """测试表相关操作"""
        async with self.get_mysql_session(use_root=True, database=test_database) as session:
            # 创建临时表用于测试
            table_name = "temp_test_table"
            create_sql = f"""
            CREATE TABLE {table_name} (
                id INT PRIMARY KEY,
                data VARCHAR(100)
            )
            """
            await session.call_tool("exec", {"sql": create_sql})
            
            # 测试 show_tables
            result = await session.call_tool("show_tables", {})
            assert table_name in result.content[0].text
            
            # 测试 describe_table
            result = await session.call_tool("describe_table", {"table": table_name})
            assert "id" in result.content[0].text
            assert "data" in result.content[0].text
            
            # 清理临时表
            await session.call_tool("exec", {"sql": f"DROP TABLE {table_name}"})
            print("✅ 表操作测试通过")

    @pytest.mark.asyncio
    async def test_security_flags(self):
        """测试安全标志功能"""
        # 启动带有 --disable-drop 标志的服务器
        server_params = StdioServerParameters(
            command=self.config["mcp_binary"],
            args=["mysql", "--disable-drop"],
        )
        
        async with stdio_client(server_params) as (read, write):
            async with ClientSession(read, write) as session:
                await session.initialize()
                
                # 连接数据库
                connect_args = {
                    "host": self.config["mysql"]["host"],
                    "port": self.config["mysql"]["port"],
                    "user": self.config["mysql"]["root_user"],
                    "password": self.config["mysql"]["root_password"],
                }
                await session.call_tool("connect_mysql", connect_args)
                
                # 尝试执行 DROP 操作（应该失败）
                result = await session.call_tool("exec", {
                    "sql": "DROP TABLE IF EXISTS non_existent_table"
                })
                content = result.content[0].text
                assert "DROP" in content and "禁用" in content
                print("✅ 安全标志测试通过")


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-s"])