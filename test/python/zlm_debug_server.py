#!/usr/bin/env python3
"""
ZLM 调试 WebSocket 服务器
用于远程调试 RTP 流传输问题

功能：
1. WebSocket 服务器，接收调试命令
2. 白名单命令配置（支持通配符 *）
3. 实时日志输出
4. 网络抓包
5. 流状态监控

使用方法：
1. 修改配置：编辑底部的 CONFIG 部分
2. 启动服务：python zlm_debug_server.py
3. 客户端连接：ws://ZLM_IP:WEBSOCKET_PORT
4. 发送命令：{"cmd": "命令", "args": {}}
"""

import asyncio
import json
import subprocess
import fnmatch
import logging
import os
import sys
from datetime import datetime
from typing import Dict, List, Optional
import websockets
from websockets.server import WebSocketServerProtocol
import shlex
from aiohttp import web
import socket


class CommandWhitelist:
    """命令白名单管理器"""

    def __init__(self, patterns: List[str]):
        """
        初始化白名单

        Args:
            patterns: 允许的命令模式列表（支持 * 通配符）
                     例如：["tcpdump*", "curl*", "netstat*", "ls*", "cat /var/log/*"]
        """
        self.patterns = patterns
        self.logger = logging.getLogger("Whitelist")

    def is_allowed(self, command: str) -> bool:
        """
        检查命令是否在白名单中

        Args:
            command: 要检查的完整命令

        Returns:
            是否允许执行
        """
        # 提取命令的第一部分（程序名）
        cmd_parts = command.strip().split()
        if not cmd_parts:
            return False

        cmd_name = cmd_parts[0]

        # 检查完整命令和命令名
        for pattern in self.patterns:
            # 支持完整命令匹配
            if fnmatch.fnmatch(command, pattern):
                self.logger.debug(f"命令匹配模式（完整）: {pattern}")
                return True

            # 支持命令名匹配
            if fnmatch.fnmatch(cmd_name, pattern):
                self.logger.debug(f"命令匹配模式（名称）: {pattern}")
                return True

        return False

    def add_pattern(self, pattern: str):
        """添加新的白名单模式"""
        if pattern not in self.patterns:
            self.patterns.append(pattern)
            self.logger.info(f"添加白名单模式: {pattern}")

    def remove_pattern(self, pattern: str):
        """移除白名单模式"""
        if pattern in self.patterns:
            self.patterns.remove(pattern)
            self.logger.info(f"移除白名单模式: {pattern}")

    def list_patterns(self) -> List[str]:
        """列出所有白名单模式"""
        return self.patterns.copy()


class CommandExecutor:
    """命令执行器"""

    def __init__(self, whitelist: CommandWhitelist, timeout: int = 30):
        """
        初始化命令执行器

        Args:
            whitelist: 命令白名单管理器
            timeout: 命令执行超时时间（秒）
        """
        self.whitelist = whitelist
        self.timeout = timeout
        self.logger = logging.getLogger("Executor")

    async def execute(self, command: str) -> Dict:
        """
        执行命令

        Args:
            command: 要执行的命令

        Returns:
            {
                "success": bool,
                "output": str,
                "error": str,
                "returncode": int,
                "duration": float
            }
        """
        # 安全检查：检测危险的 shell 元字符
        dangerous_chars = [';', '&&', '||', '|', '$', '`', '>', '<', '\n', '\r', '!']
        for char in dangerous_chars:
            if char in command:
                error_msg = f"命令包含危险的 shell 元字符: '{char}'"
                self.logger.warning(error_msg)
                return {
                    "success": False,
                    "output": "",
                    "error": error_msg,
                    "returncode": -1,
                    "duration": 0.0
                }

        # 检查白名单
        if not self.whitelist.is_allowed(command):
            error_msg = f"命令不在白名单中: {command}"
            self.logger.warning(error_msg)
            return {
                "success": False,
                "output": "",
                "error": error_msg,
                "returncode": -1,
                "duration": 0.0
            }

        # 记录执行
        self.logger.info(f"执行命令: {command}")
        start_time = datetime.now()

        try:
            # 异步执行命令
            process = await asyncio.create_subprocess_shell(
                command,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )

            # 等待命令完成（带超时）
            try:
                stdout, stderr = await asyncio.wait_for(
                    process.communicate(),
                    timeout=self.timeout
                )
            except asyncio.TimeoutError:
                process.kill()
                error_msg = f"命令执行超时（{self.timeout}秒）"
                self.logger.error(error_msg)
                return {
                    "success": False,
                    "output": "",
                    "error": error_msg,
                    "returncode": -1,
                    "duration": self.timeout
                }

            # 计算执行时间
            duration = (datetime.now() - start_time).total_seconds()

            # 解码输出
            output = stdout.decode('utf-8', errors='replace')
            error = stderr.decode('utf-8', errors='replace')

            # 构建结果
            result = {
                "success": process.returncode == 0,
                "output": output,
                "error": error,
                "returncode": process.returncode,
                "duration": duration
            }

            self.logger.info(f"命令完成: 返回码={process.returncode}, 耗时={duration:.2f}秒")
            return result

        except Exception as e:
            error_msg = f"命令执行异常: {str(e)}"
            self.logger.error(error_msg, exc_info=True)
            return {
                "success": False,
                "output": "",
                "error": error_msg,
                "returncode": -1,
                "duration": 0.0
            }


class DebugServer:
    """调试 WebSocket 服务器"""

    def __init__(self, host: str, port: int, whitelist_patterns: List[str],
                 command_timeout: int = 30, http_port: Optional[int] = None):
        """
        初始化调试服务器

        Args:
            host: 监听地址
            port: WebSocket 监听端口
            whitelist_patterns: 命令白名单模式
            command_timeout: 命令执行超时
            http_port: HTTP API 端口（None 表示不启动 HTTP 服务）
        """
        self.host = host
        self.port = port
        self.http_port = http_port
        self.whitelist = CommandWhitelist(whitelist_patterns)
        self.executor = CommandExecutor(self.whitelist, command_timeout)
        self.logger = logging.getLogger("DebugServer")
        self.clients: Dict[str, WebSocketServerProtocol] = {}
        self.http_runner: Optional[web.AppRunner] = None

    async def handle_client(self, websocket: WebSocketServerProtocol, path: str):
        """处理客户端连接"""
        client_id = f"{websocket.remote_address[0]}:{websocket.remote_address[1]}"
        self.clients[client_id] = websocket
        self.logger.info(f"客户端连接: {client_id} (当前连接数: {len(self.clients)})")

        try:
            # 发送欢迎消息
            await websocket.send(json.dumps({
                "type": "welcome",
                "message": f"连接成功，客户端ID: {client_id}",
                "whitelist": self.whitelist.list_patterns(),
                "timestamp": datetime.now().isoformat()
            }))

            # 处理消息
            async for message in websocket:
                try:
                    # 解析消息
                    data = json.loads(message)
                    await self.handle_message(websocket, data)
                except json.JSONDecodeError as e:
                    await websocket.send(json.dumps({
                        "type": "error",
                        "message": f"JSON 解析错误: {str(e)}",
                        "timestamp": datetime.now().isoformat()
                    }))
                except Exception as e:
                    self.logger.error(f"处理消息异常: {str(e)}", exc_info=True)
                    await websocket.send(json.dumps({
                        "type": "error",
                        "message": f"处理消息异常: {str(e)}",
                        "timestamp": datetime.now().isoformat()
                    }))

        except websockets.exceptions.ConnectionClosed:
            self.logger.info(f"客户端断开连接: {client_id}")
        except Exception as e:
            self.logger.error(f"客户端连接异常: {str(e)}", exc_info=True)
        finally:
            del self.clients[client_id]
            self.logger.info(f"客户端清理完成: {client_id} (当前连接数: {len(self.clients)})")

    async def handle_message(self, websocket: WebSocketServerProtocol, data: Dict):
        """处理客户端消息"""
        msg_type = data.get("type", "command")

        if msg_type == "command":
            # 执行命令
            command = data.get("cmd") or data.get("command")
            if not command:
                await websocket.send(json.dumps({
                    "type": "error",
                    "message": "缺少命令参数 'cmd' 或 'command'",
                    "timestamp": datetime.now().isoformat()
                }))
                return

            # 执行并返回结果
            result = await self.executor.execute(command)
            await websocket.send(json.dumps({
                "type": "command_result",
                "command": command,
                "result": result,
                "timestamp": datetime.now().isoformat()
            }))

        elif msg_type == "add_whitelist":
            # 已禁用动态添加白名单（安全考虑）
            await websocket.send(json.dumps({
                "type": "error",
                "message": "动态添加白名单功能已被禁用，请修改配置文件重启服务器",
                "timestamp": datetime.now().isoformat()
            }))

        elif msg_type == "remove_whitelist":
            # 已禁用动态移除白名单（安全考虑）
            await websocket.send(json.dumps({
                "type": "error",
                "message": "动态移除白名单功能已被禁用，请修改配置文件重启服务器",
                "timestamp": datetime.now().isoformat()
            }))

        elif msg_type == "list_whitelist":
            # 列出白名单
            await websocket.send(json.dumps({
                "type": "whitelist_list",
                "whitelist": self.whitelist.list_patterns(),
                "timestamp": datetime.now().isoformat()
            }))

        elif msg_type == "ping":
            # 心跳
            await websocket.send(json.dumps({
                "type": "pong",
                "timestamp": datetime.now().isoformat()
            }))

        else:
            await websocket.send(json.dumps({
                "type": "error",
                "message": f"未知的消息类型: {msg_type}",
                "timestamp": datetime.now().isoformat()
            }))

    async def broadcast(self, message: Dict):
        """广播消息给所有客户端"""
        if self.clients:
            await asyncio.gather(
                *[client.send(json.dumps(message)) for client in self.clients.values()]
            )

    # ==================== HTTP API ====================

    async def handle_http_get_whitelist(self, request: web.Request):
        """HTTP API: 获取白名单列表"""
        return web.json_response({
            "success": True,
            "whitelist": self.whitelist.list_patterns(),
            "count": len(self.whitelist.list_patterns()),
            "timestamp": datetime.now().isoformat()
        })

    async def handle_http_add_whitelist(self, request: web.Request):
        """HTTP API: 添加白名单模式（已禁用）"""
        return web.json_response({
            "success": False,
            "error": "动态添加白名单功能已被禁用，请修改配置文件重启服务器"
        }, status=403)

    async def handle_http_remove_whitelist(self, request: web.Request):
        """HTTP API: 移除白名单模式（已禁用）"""
        return web.json_response({
            "success": False,
            "error": "动态移除白名单功能已被禁用，请修改配置文件重启服务器"
        }, status=403)

    async def handle_http_check_command(self, request: web.Request):
        """HTTP API: 检查命令是否在白名单中"""
        try:
            data = await request.json()
            command = data.get("command")

            if not command:
                return web.json_response({
                    "success": False,
                    "error": "缺少参数 'command'"
                }, status=400)

            is_allowed = self.whitelist.is_allowed(command)

            return web.json_response({
                "success": True,
                "command": command,
                "allowed": is_allowed,
                "timestamp": datetime.now().isoformat()
            })
        except Exception as e:
            return web.json_response({
                "success": False,
                "error": str(e)
            }, status=500)

    async def handle_http_status(self, request: web.Request):
        """HTTP API: 获取服务器状态"""
        return web.json_response({
            "success": True,
            "status": "running",
            "websocket": {
                "host": self.host,
                "port": self.port,
                "clients": len(self.clients)
            },
            "http_port": self.http_port,
            "whitelist_count": len(self.whitelist.list_patterns()),
            "command_timeout": self.executor.timeout,
            "timestamp": datetime.now().isoformat()
        })

    async def handle_http_execute(self, request: web.Request):
        """HTTP API: 执行命令（通过 HTTP）"""
        try:
            data = await request.json()
            command = data.get("command")

            if not command:
                return web.json_response({
                    "success": False,
                    "error": "缺少参数 'command'"
                }, status=400)

            # 执行命令
            result = await self.executor.execute(command)

            return web.json_response({
                "success": True,
                "command": command,
                "result": result,
                "timestamp": datetime.now().isoformat()
            })
        except Exception as e:
            return web.json_response({
                "success": False,
                "error": str(e)
            }, status=500)

    async def start_http_server(self):
        """启动 HTTP API 服务器"""
        if not self.http_port:
            return

        app = web.Application()

        # 注册路由
        app.router.add_get('/api/whitelist', self.handle_http_get_whitelist)
        # 已禁用动态修改白名单的接口（安全考虑）
        # app.router.add_post('/api/whitelist/add', self.handle_http_add_whitelist)
        # app.router.add_post('/api/whitelist/remove', self.handle_http_remove_whitelist)
        app.router.add_post('/api/command/check', self.handle_http_check_command)
        app.router.add_post('/api/command/execute', self.handle_http_execute)
        app.router.add_get('/api/status', self.handle_http_status)

        # 启动服务
        self.http_runner = web.AppRunner(app)
        await self.http_runner.setup()

        site = web.TCPSite(self.http_runner, self.host, self.http_port)
        await site.start()

        self.logger.info(f"HTTP API 服务器已启动: http://{self.host}:{self.http_port}")

    async def stop_http_server(self):
        """停止 HTTP API 服务器"""
        if self.http_runner:
            await self.http_runner.cleanup()
            self.logger.info("HTTP API 服务器已停止")

    def run(self):
        """启动服务器"""
        self.logger.info(f"启动调试服务器: ws://{self.host}:{self.port}")
        if self.http_port:
            self.logger.info(f"HTTP API 端口: {self.http_port}")
        self.logger.info(f"白名单模式: {self.whitelist.list_patterns()}")

        async def start_all():
            # 启动 HTTP API 服务
            await self.start_http_server()

            # 启动 WebSocket 服务
            async with websockets.serve(
                self.handle_client,
                self.host,
                self.port,
                ping_interval=20,
                ping_timeout=10
            ):
                # 保持运行
                await asyncio.Future()  # 永久运行

        try:
            asyncio.run(start_all())
        except KeyboardInterrupt:
            self.logger.info("收到中断信号，正在关闭...")


# ==================== 配置区域 ====================

CONFIG = {
    # WebSocket 服务器配置
    "host": "0.0.0.0",  # 监听所有网卡
    "port": 8999,       # WebSocket 端口

    # HTTP API 服务器配置
    "http_port": 8998,  # HTTP API 端口（None 表示不启动）

    # 命令执行配置
    "command_timeout": 600,  # 命令执行超时（秒）

    # 命令白名单（支持 * 通配符）
    # 格式：
    # - "命令名*"：允许该命令的所有参数
    # - "命令名 参数*"：允许特定参数模式
    # - "完整命令"：只允许这个精确命令
    "whitelist": [
    "curl *",
    "ss *",
    "netstat *",    
    # 网络工具 - 更精确的模式
    "sudo tcpdump -i ens18 *",         # 只允许指定接口参数
    "sudo tcpdump",              # 只允许无参数的 tcpdump
    "iftop  -i ens18 *",                # 允许 iftop
    
    # 文件查看 - 限制路径
    "ls",                   # 无参数的 ls
    "ls -la",               # 只允许 -la 参数
    "ls /var/log/*",        # 只允许查看日志目录
    "ls /tmp/*",            # 只允许查看临时目录
    "cat /var/log/*",       # 已有，保留
    "cat /tmp/*",           # 已有，保留
    "tail -f /tmp/*",   # 只允许跟踪日志文件
    "tail -f /var/log/*",   # 只允许跟踪日志文件
    "tail /var/log/*",      # 只允许查看日志文件
    "head /var/log/*",      # 只允许查看日志文件头部
    "less /var/log/*",      # 只允许查看日志文件
    "more /var/log/*",      # 只允许查看日志文件
    
    "docker logs * zlmediakit-zlmediakit-1",
    "docker logs -f zlmediakit-zlmediakit-1",
    "docker logs -f zlmediakit-zlmediakit-1 *"
    ],

    # 日志配置
    "log_level": "INFO",  # DEBUG, INFO, WARNING, ERROR
    "log_format": "%(asctime)s [%(levelname)s] %(name)s: %(message)s",
}

# ==================== 主程序 ====================

def setup_logging():
    """配置日志"""
    logging.basicConfig(
        level=getattr(logging, CONFIG["log_level"]),
        format=CONFIG["log_format"],
        handlers=[
            logging.StreamHandler(sys.stdout),
        ]
    )

    # 降低第三方库的日志级别
    logging.getLogger("websockets").setLevel(logging.WARNING)


def main():
    """主函数"""
    setup_logging()
    logger = logging.getLogger("Main")

    logger.info("="*60)
    logger.info("ZLM 调试 WebSocket 服务器")
    logger.info("="*60)
    logger.info(f"WebSocket 地址: ws://{CONFIG['host']}:{CONFIG['port']}")
    if CONFIG.get('http_port'):
        logger.info(f"HTTP API 地址: http://{CONFIG['host']}:{CONFIG['http_port']}")
    logger.info(f"命令超时: {CONFIG['command_timeout']} 秒")
    logger.info(f"白名单模式数量: {len(CONFIG['whitelist'])}")
    logger.info("="*60)

    # 创建并启动服务器
    server = DebugServer(
        host=CONFIG["host"],
        port=CONFIG["port"],
        whitelist_patterns=CONFIG["whitelist"],
        command_timeout=CONFIG["command_timeout"],
        http_port=CONFIG.get("http_port")
    )

    try:
        server.run()
    except KeyboardInterrupt:
        logger.info("\n收到中断信号，正在关闭服务器...")
    except Exception as e:
        logger.error(f"服务器异常: {str(e)}", exc_info=True)
    finally:
        logger.info("服务器已关闭")


if __name__ == "__main__":
    main()