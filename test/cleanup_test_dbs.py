#!/usr/bin/env python3
"""
清理残留的测试数据库

扫描并删除所有以 test_mcp_ 开头的数据库
"""

import asyncio
import yaml
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


async def cleanup_test_databases():
    """清理所有测试数据库"""
    # 加载配置
    with open('config.yaml', 'r') as f:
        config = yaml.safe_load(f)
    
    # 创建服务器参数
    server_params = StdioServerParameters(
        command=config["mcp_binary"],
        args=["mysql"],
    )
    
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            
            # 使用 root 用户连接
            connect_args = {
                "host": config["mysql"]["host"],
                "port": config["mysql"]["port"],
                "user": config["mysql"]["root_user"],
                "password": config["mysql"]["root_password"],
            }
            
            result = await session.call_tool("connect_mysql", connect_args)
            print(f"连接状态: {result.content[0].text[:50]}...")
            
            # 获取所有数据库
            result = await session.call_tool("exec", {"sql": "SHOW DATABASES"})
            databases_text = result.content[0].text
            
            # 查找测试数据库
            test_dbs = []
            for line in databases_text.split('\n'):
                if 'test_mcp_' in line:
                    # 提取数据库名
                    db_name = line.strip().split('|')[0].strip() if '|' in line else line.strip()
                    if db_name.startswith('test_mcp_'):
                        test_dbs.append(db_name)
            
            if not test_dbs:
                print("✅ 没有发现残留的测试数据库")
                return
            
            print(f"\n发现 {len(test_dbs)} 个测试数据库:")
            for db in test_dbs:
                print(f"  - {db}")
            
            # 询问确认
            confirm = input("\n是否删除这些数据库? (yes/no): ")
            if confirm.lower() != 'yes':
                print("取消操作")
                return
            
            # 删除数据库
            for db in test_dbs:
                try:
                    result = await session.call_tool("exec", {
                        "sql": f"DROP DATABASE {db}"
                    })
                    print(f"✅ 已删除: {db}")
                except Exception as e:
                    print(f"❌ 删除 {db} 失败: {e}")
            
            print("\n清理完成！")


if __name__ == "__main__":
    asyncio.run(cleanup_test_databases())