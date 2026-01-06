#!/usr/bin/env python3
"""SQLite MCP 工具测试脚本"""

import asyncio
import sys
from pathlib import Path

# 添加父目录到 Python 路径
sys.path.insert(0, str(Path(__file__).parent))

from mcp_client import MCPClient


async def test_sqlite():
    """测试 SQLite 功能"""
    print("=" * 60)
    print("SQLite MCP 工具测试")
    print("=" * 60)

    client = MCPClient()

    try:
        await client.connect()

        # 测试 1: 内存数据库 - SELECT 查询
        print("\n测试 1: 内存数据库 SELECT 查询")
        print("-" * 60)
        result = await client.call_tool(
            "sqlite_exec",
            dsn=":memory:",
            sql="SELECT 1 as test_value, 'SQLite is working!' as message"
        )
        print(result)

        # 测试 2: 内存数据库 - 创建表
        print("\n测试 2: 创建表")
        print("-" * 60)
        result = await client.call_tool(
            "sqlite_exec",
            dsn=":memory:",
            sql="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"
        )
        print(result)

        # 测试 3: 文件数据库 - 创建并插入数据
        db_path = "/tmp/test_mcp.db"
        print(f"\n测试 3: 文件数据库 ({db_path})")
        print("-" * 60)

        # 创建表
        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="CREATE TABLE IF NOT EXISTS products (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, price REAL)"
        )
        print(result)

        # 插入数据
        print("\n插入数据...")
        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="INSERT INTO products (name, price) VALUES ('Laptop', 999.99)"
        )
        print(result)

        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="INSERT INTO products (name, price) VALUES ('Mouse', 29.99)"
        )
        print(result)

        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="INSERT INTO products (name, price) VALUES ('Keyboard', 79.99)"
        )
        print(result)

        # 查询数据
        print("\n查询所有数据...")
        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="SELECT * FROM products"
        )
        print(result)

        # 更新数据
        print("\n更新数据...")
        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="UPDATE products SET price = 899.99 WHERE name = 'Laptop'"
        )
        print(result)

        # 删除数据
        print("\n删除数据...")
        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="DELETE FROM products WHERE name = 'Mouse'"
        )
        print(result)

        # 查询最终结果
        print("\n查询最终结果...")
        result = await client.call_tool(
            "sqlite_exec",
            dsn=db_path,
            sql="SELECT * FROM products ORDER BY id"
        )
        print(result)

        print("\n" + "=" * 60)
        print("✓ 所有测试通过!")
        print("=" * 60)

    finally:
        await client.close()


if __name__ == "__main__":
    asyncio.run(test_sqlite())
