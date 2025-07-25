"""
MySQL Provider 集成测试

测试完整的数据库生命周期：
1. 创建测试数据库
2. 创建测试表
3. 插入数据 (INSERT)
4. 查询数据 (SELECT)
5. 更新数据 (UPDATE)
6. 删除数据 (DELETE)
7. 删除表 (DROP TABLE)
8. 删除数据库 (DROP DATABASE)
"""

import asyncio
import yaml
import pytest
import uuid
from contextlib import asynccontextmanager
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


class TestMySQL:
    @classmethod
    def setup_class(cls):
        """加载配置"""
        with open('config.yaml', 'r') as f:
            cls.config = yaml.safe_load(f)
        
        # 生成唯一的测试数据库名
        cls.test_db_name = f"test_mcp_{uuid.uuid4().hex[:8]}"
        cls.test_table_name = "test_users"
        print(f"使用测试数据库: {cls.test_db_name}")
    
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
                    # 使用 root 用户（用于创建/删除数据库）
                    connect_args = {
                        "host": self.config["mysql"]["host"],
                        "port": self.config["mysql"]["port"],
                        "user": self.config["mysql"]["root_user"],
                        "password": self.config["mysql"]["root_password"],
                    }
                else:
                    # 使用普通用户
                    connect_args = {
                        "host": self.config["mysql"]["host"],
                        "port": self.config["mysql"]["port"],
                        "user": self.config["mysql"]["user"],
                        "password": self.config["mysql"]["password"],
                    }
                
                if database:
                    connect_args["database"] = database
                    
                result = await session.call_tool("connect_mysql", connect_args)
                
                # 验证连接成功（兼容不同的返回格式）
                content = result.content[0].text
                assert any(success_indicator in content for success_indicator in 
                          ["✅", "成功", "Connected", "successfully"]), f"连接失败: {content}"
                
                yield session

    @pytest.mark.asyncio
    async def test_01_connection_info(self):
        """测试连接信息显示"""
        async with self.get_mysql_session(use_root=True) as session:
            # 测试当前连接信息
            result = await session.call_tool("current_mysql", {})
            content = result.content[0].text
            assert "47.119.182.61" in content
            assert "15640" in content
            print(f"当前连接信息: {content}")

    @pytest.mark.asyncio
    async def test_02_create_database(self):
        """测试创建数据库"""
        async with self.get_mysql_session(use_root=True) as session:
            # 创建测试数据库
            result = await session.call_tool("exec", {
                "sql": f"CREATE DATABASE {self.test_db_name} CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
            })
            
            # 验证创建成功
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["Query OK", "执行成功", "success", "1 row affected"]), f"创建数据库失败: {content}"
            
            # 验证数据库存在
            result = await session.call_tool("exec", {
                "sql": "SHOW DATABASES"
            })
            assert self.test_db_name in result.content[0].text
            print(f"数据库 {self.test_db_name} 创建成功")

    @pytest.mark.asyncio 
    async def test_03_create_table(self):
        """测试创建表"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 创建测试表
            create_table_sql = f"""
            CREATE TABLE {self.test_table_name} (
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
            assert any(success_indicator in content for success_indicator in 
                      ["Query OK", "执行成功", "success"]), f"创建表失败: {content}"
            
            # 验证表结构
            result = await session.call_tool("describe_table", {"table": self.test_table_name})
            table_info = result.content[0].text
            assert "name" in table_info
            assert "email" in table_info
            assert "用户姓名" in table_info or "name" in table_info
            print(f"表 {self.test_table_name} 创建成功")

    @pytest.mark.asyncio
    async def test_04_show_tables(self):
        """测试显示表信息"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 测试 show_tables 工具
            result = await session.call_tool("show_tables", {})
            content = result.content[0].text
            assert self.test_table_name in content
            assert any(indicator in content for indicator in ["InnoDB", "MyISAM", "Engine"])
            print(f"表信息显示成功: {self.test_table_name}")

    @pytest.mark.asyncio
    async def test_05_insert_data(self):
        """测试插入数据"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 插入单条记录
            result = await session.call_tool("exec", {
                "sql": f"INSERT INTO {self.test_table_name} (name, email, age) VALUES ('张三', 'zhangsan@test.com', 25)"
            })
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["1 row", "影响行数: 1", "affected", "Query OK"]), f"插入失败: {content}"
            
            # 批量插入
            batch_insert_sql = f"""
            INSERT INTO {self.test_table_name} (name, email, age) VALUES 
            ('李四', 'lisi@test.com', 30),
            ('王五', 'wangwu@test.com', 28),
            ('赵六', 'zhaoliu@test.com', 35)
            """
            
            result = await session.call_tool("exec", {"sql": batch_insert_sql})
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["3 row", "影响行数: 3", "affected", "Query OK"]), f"批量插入失败: {content}"
            
            # 验证插入结果
            result = await session.call_tool("exec", {
                "sql": f"SELECT COUNT(*) as total FROM {self.test_table_name}"
            })
            assert "4" in result.content[0].text  # 总共4条记录
            print("数据插入成功，共4条记录")

    @pytest.mark.asyncio
    async def test_06_select_data(self):
        """测试查询数据"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 查询所有数据
            result = await session.call_tool("exec", {
                "sql": f"SELECT * FROM {self.test_table_name} ORDER BY id"
            })
            content = result.content[0].text
            assert "张三" in content
            assert "zhangsan@test.com" in content
            
            # 条件查询
            result = await session.call_tool("exec", {
                "sql": f"SELECT name, age FROM {self.test_table_name} WHERE age > 30"
            })
            content = result.content[0].text
            assert "赵六" in content
            assert "35" in content
            
            # 聚合查询
            result = await session.call_tool("exec", {
                "sql": f"SELECT AVG(age) as avg_age, COUNT(*) as total FROM {self.test_table_name}"
            })
            content = result.content[0].text
            assert any(avg in content for avg in ["29.5", "29"]) # 平均年龄
            assert "4" in content  # 总数
            print("数据查询验证成功")

    @pytest.mark.asyncio
    async def test_07_update_data(self):
        """测试更新数据"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 更新单条记录
            result = await session.call_tool("exec", {
                "sql": f"UPDATE {self.test_table_name} SET age = 26 WHERE name = '张三'"
            })
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["1 row", "影响行数: 1", "affected", "Query OK"]), f"更新失败: {content}"
            
            # 批量更新
            result = await session.call_tool("exec", {
                "sql": f"UPDATE {self.test_table_name} SET age = age + 1 WHERE age < 30"
            })
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["row", "affected", "Query OK"]), f"批量更新失败: {content}"
            
            # 验证更新结果
            result = await session.call_tool("exec", {
                "sql": f"SELECT name, age FROM {self.test_table_name} WHERE name = '张三'"
            })
            content = result.content[0].text
            assert "27" in content  # 26 + 1 = 27
            print("数据更新验证成功")

    @pytest.mark.asyncio
    async def test_08_delete_data(self):
        """测试删除数据"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 删除单条记录
            result = await session.call_tool("exec", {
                "sql": f"DELETE FROM {self.test_table_name} WHERE name = '王五'"
            })
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["1 row", "影响行数: 1", "affected", "Query OK"]), f"删除失败: {content}"
            
            # 验证删除结果
            result = await session.call_tool("exec", {
                "sql": f"SELECT COUNT(*) as total FROM {self.test_table_name}"
            })
            assert "3" in result.content[0].text  # 剩余3条记录
            
            # 条件删除
            result = await session.call_tool("exec", {
                "sql": f"DELETE FROM {self.test_table_name} WHERE age > 35"
            })
            # 验证最终记录数
            result = await session.call_tool("exec", {
                "sql": f"SELECT COUNT(*) as total FROM {self.test_table_name}"
            })
            print(f"删除后剩余记录数: {result.content[0].text}")

    @pytest.mark.asyncio
    async def test_09_describe_tables(self):
        """测试批量描述表"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 测试 describe_tables 工具
            result = await session.call_tool("describe_tables", {
                "tables": self.test_table_name,
                "include_indexes": True,
                "include_foreign_keys": True
            })
            content = result.content[0].text
            assert any(key_info in content for key_info in ["PRIMARY", "AUTO_INCREMENT", "UNIQUE"])
            print("表结构描述成功")

    @pytest.mark.asyncio
    async def test_10_drop_table(self):
        """测试删除表"""
        async with self.get_mysql_session(use_root=True, database=self.test_db_name) as session:
            # 删除表
            result = await session.call_tool("exec", {
                "sql": f"DROP TABLE {self.test_table_name}"
            })
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["Query OK", "执行成功", "success"]), f"删除表失败: {content}"
            
            # 验证表已删除
            result = await session.call_tool("show_tables", {})
            assert self.test_table_name not in result.content[0].text
            print(f"表 {self.test_table_name} 删除成功")

    @pytest.mark.asyncio
    async def test_11_drop_database(self):
        """测试删除数据库（清理）"""
        async with self.get_mysql_session(use_root=True) as session:
            # 删除测试数据库
            result = await session.call_tool("exec", {
                "sql": f"DROP DATABASE {self.test_db_name}"
            })
            content = result.content[0].text
            assert any(success_indicator in content for success_indicator in 
                      ["Query OK", "执行成功", "success"]), f"删除数据库失败: {content}"
            
            # 验证数据库已删除
            result = await session.call_tool("exec", {
                "sql": "SHOW DATABASES"
            })
            assert self.test_db_name not in result.content[0].text
            print(f"数据库 {self.test_db_name} 删除成功，测试清理完成")


if __name__ == "__main__":
    # 直接运行测试
    pytest.main([__file__, "-v", "-s"])