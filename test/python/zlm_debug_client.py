#!/usr/bin/env python3
"""
ZLM 调试 WebSocket 客户端
用于连接部署在 ZLM 端的调试服务器

使用方法：
1. 启动服务：python zlm_debug_client.py
2. 输入命令或使用快捷命令
"""

import asyncio
import json
import logging
import sys
from datetime import datetime
from typing import Optional
import websockets
from websockets.client import WebSocketClientProtocol


class DebugClient:
    """调试 WebSocket 客户端"""

    def __init__(self, server_url: str):
        """
        初始化客户端

        Args:
            server_url: WebSocket 服务器地址，如 ws://10.10.10.200:8999
        """
        self.server_url = server_url
        self.websocket: Optional[WebSocketClientProtocol] = None
        self.logger = logging.getLogger("DebugClient")
        self.running = False

    async def connect(self):
        """连接到服务器"""
        self.logger.info(f"连接到服务器: {self.server_url}")
        try:
            self.websocket = await websockets.connect(
                self.server_url,
                ping_interval=20,
                ping_timeout=10
            )
            self.running = True
            self.logger.info("连接成功")

            # 启动接收任务
            receive_task = asyncio.create_task(self.receive_messages())

            # 等待欢迎消息
            welcome = await self.websocket.recv()
            welcome_data = json.loads(welcome)
            self.print_message(welcome_data)

            return True
        except Exception as e:
            self.logger.error(f"连接失败: {str(e)}")
            return False

    async def receive_messages(self):
        """接收服务器消息"""
        try:
            async for message in self.websocket:
                data = json.loads(message)
                self.print_message(data)
        except websockets.exceptions.ConnectionClosed:
            self.logger.info("连接已关闭")
            self.running = False
        except Exception as e:
            self.logger.error(f"接收消息异常: {str(e)}", exc_info=True)
            self.running = False

    async def send_command(self, command: str):
        """发送命令"""
        if not self.websocket or not self.running:
            self.logger.error("未连接到服务器")
            return

        try:
            await self.websocket.send(json.dumps({
                "type": "command",
                "cmd": command
            }))
        except Exception as e:
            self.logger.error(f"发送命令失败: {str(e)}")

    async def add_whitelist(self, pattern: str):
        """添加白名单模式"""
        if not self.websocket or not self.running:
            self.logger.error("未连接到服务器")
            return

        try:
            await self.websocket.send(json.dumps({
                "type": "add_whitelist",
                "pattern": pattern
            }))
        except Exception as e:
            self.logger.error(f"添加白名单失败: {str(e)}")

    async def remove_whitelist(self, pattern: str):
        """移除白名单模式"""
        if not self.websocket or not self.running:
            self.logger.error("未连接到服务器")
            return

        try:
            await self.websocket.send(json.dumps({
                "type": "remove_whitelist",
                "pattern": pattern
            }))
        except Exception as e:
            self.logger.error(f"移除白名单失败: {str(e)}")

    async def list_whitelist(self):
        """列出白名单"""
        if not self.websocket or not self.running:
            self.logger.error("未连接到服务器")
            return

        try:
            await self.websocket.send(json.dumps({
                "type": "list_whitelist"
            }))
        except Exception as e:
            self.logger.error(f"列出白名单失败: {str(e)}")

    async def close(self):
        """关闭连接"""
        if self.websocket:
            await self.websocket.close()
            self.logger.info("连接已关闭")

    def print_message(self, data: dict):
        """格式化打印消息"""
        msg_type = data.get("type", "unknown")
        timestamp = data.get("timestamp", datetime.now().isoformat())

        print(f"\n{'='*60}")
        print(f"[{timestamp}] 类型: {msg_type}")
        print(f"{'='*60}")

        if msg_type == "welcome":
            print(f"消息: {data.get('message')}")
            print(f"\n当前白名单模式:")
            for pattern in data.get('whitelist', []):
                print(f"  - {pattern}")

        elif msg_type == "command_result":
            command = data.get('command')
            result = data.get('result', {})
            print(f"命令: {command}")
            print(f"执行时间: {result.get('duration', 0):.2f} 秒")
            print(f"返回码: {result.get('returncode')}")
            print(f"成功: {'✓' if result.get('success') else '✗'}")

            if result.get('output'):
                print(f"\n输出:\n{result['output']}")

            if result.get('error'):
                print(f"\n错误:\n{result['error']}")

        elif msg_type == "whitelist_updated":
            print(f"消息: {data.get('message')}")
            print(f"\n当前白名单模式:")
            for pattern in data.get('whitelist', []):
                print(f"  - {pattern}")

        elif msg_type == "whitelist_list":
            print(f"当前白名单模式:")
            for pattern in data.get('whitelist', []):
                print(f"  - {pattern}")

        elif msg_type == "error":
            print(f"错误: {data.get('message')}")

        else:
            print(json.dumps(data, indent=2, ensure_ascii=False))

        print(f"{'='*60}\n")


async def interactive_shell(client: DebugClient):
    """交互式命令行"""
    print("\n" + "="*60)
    print("ZLM 调试客户端 - 交互式命令行")
    print("="*60)
    print("命令:")
    print("  /help              - 显示帮助")
    print("  /whitelist         - 列出白名单")
    print("  /add <pattern>     - 添加白名单模式")
    print("  /remove <pattern>  - 移除白名单模式")
    print("  /exit              - 退出")
    print("  <命令>             - 执行系统命令")
    print("="*60)
    print()

    while client.running:
        try:
            # 读取用户输入
            user_input = await asyncio.get_event_loop().run_in_executor(
                None,
                input,
                "调试> "
            )

            user_input = user_input.strip()
            if not user_input:
                continue

            # 处理特殊命令
            if user_input == "/help":
                print("\n快捷命令:")
                print("  /help              - 显示帮助")
                print("  /whitelist         - 列出白名单")
                print("  /add <pattern>     - 添加白名单模式")
                print("  /remove <pattern>  - 移除白名单模式")
                print("  /exit              - 退出")
                print("\n示例:")
                print("  调试> tcpdump -i any udp port 51874")
                print("  调试> netstat -anp | grep 51874")
                print("  调试> curl http://localhost:5080/index/api/getMediaList")
                print()

            elif user_input == "/whitelist":
                await client.list_whitelist()

            elif user_input.startswith("/add "):
                pattern = user_input[5:].strip()
                if pattern:
                    await client.add_whitelist(pattern)
                    print(f"已发送添加白名单请求: {pattern}\n")
                else:
                    print("用法: /add <pattern>\n")

            elif user_input.startswith("/remove "):
                pattern = user_input[8:].strip()
                if pattern:
                    await client.remove_whitelist(pattern)
                    print(f"已发送移除白名单请求: {pattern}\n")
                else:
                    print("用法: /remove <pattern>\n")

            elif user_input == "/exit":
                print("退出...")
                break

            else:
                # 执行系统命令
                await client.send_command(user_input)

        except EOFError:
            print("\n退出...")
            break
        except KeyboardInterrupt:
            print("\n退出...")
            break
        except Exception as e:
            print(f"错误: {str(e)}")


async def main():
    """主函数"""
    # 配置
    SERVER_URL = "ws://10.10.10.200:8999"  # 修改为实际的 ZLM 服务器地址

    # 配置日志
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
        handlers=[logging.StreamHandler(sys.stdout)]
    )
    logging.getLogger("websockets").setLevel(logging.WARNING)

    # 创建客户端
    client = DebugClient(SERVER_URL)

    # 连接服务器
    if not await client.connect():
        print("连接失败，请检查服务器地址和网络")
        return

    # 启动交互式命令行
    try:
        await interactive_shell(client)
    finally:
        await client.close()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n客户端已关闭")