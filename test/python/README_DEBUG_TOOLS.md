# ZLM 调试工具使用说明

## 概述

这是一套用于调试 GB28181 RTP 流传输问题的工具，包含：
- **服务端** (`zlm_debug_server.py`): 部署在 ZLM 服务器上，提供 WebSocket 接口
- **客户端** (`zlm_debug_client.py`): 在本地运行，连接服务端执行调试命令

## 功能特性

### 服务端功能
1. ✅ WebSocket 服务器（端口可配置）
2. ✅ HTTP API 服务器（端口可配置）
3. ✅ 命令白名单机制（支持通配符 `*`）
4. ✅ 命令执行超时控制
5. ✅ 实时日志输出
6. ✅ 多客户端连接支持

### 客户端功能
1. ✅ 交互式命令行
2. ✅ 快捷命令支持
3. ✅ 格式化输出
4. ✅ 白名单管理

## 安装依赖

### 在 ZLM 服务器上安装

```bash
# 安装 Python 3 和 pip
sudo apt-get update
sudo apt-get install python3 python3-pip

# 安装依赖
pip3 install websockets aiohttp
```

### 在本地机器上安装

```bash
# Windows
pip install websockets

# Linux/Mac
pip3 install websockets
```

## 部署步骤

### 1. 配置服务端

编辑 `zlm_debug_server.py` 底部的 `CONFIG` 部分：

```python
CONFIG = {
    # WebSocket 服务器配置
    "host": "0.0.0.0",  # 监听所有网卡（建议改为 ZLM 的内网 IP）
    "port": 9999,       # WebSocket 端口

    # HTTP API 服务器配置
    "http_port": 9998,  # HTTP API 端口（None 表示不启动）

    # 命令执行配置
    "command_timeout": 60,  # 命令执行超时（秒）

    # 命令白名单
    "whitelist": [
        "tcpdump*",
        "curl*",
        "netstat*",
        # ... 根据需要添加
    ],
}
```

### 2. 启动服务端

```bash
# 前台运行
python3 zlm_debug_server.py

# 后台运行
nohup python3 zlm_debug_server.py > debug_server.log 2>&1 &

# 使用 systemd 管理（推荐）
sudo systemctl start zlmediakit-debug
```

### 3. 配置客户端

编辑 `zlm_debug_client.py` 中的服务器地址：

```python
SERVER_URL = "ws://10.10.10.200:9999"  # 修改为实际的 ZLM 服务器地址
```

### 4. 启动客户端

```bash
python3 zlm_debug_client.py
```

## HTTP API 接口

### 接口列表

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/status` | GET | 获取服务器状态 |
| `/api/whitelist` | GET | 获取白名单列表 |
| `/api/whitelist/add` | POST | 添加白名单模式 |
| `/api/whitelist/remove` | POST | 移除白名单模式 |
| `/api/command/check` | POST | 检查命令是否允许 |
| `/api/command/execute` | POST | 执行命令 |

### 使用示例

#### 1. 获取服务器状态

```bash
curl http://10.10.10.200:9998/api/status
```

响应：
```json
{
  "success": true,
  "status": "running",
  "websocket": {
    "host": "0.0.0.0",
    "port": 9999,
    "clients": 1
  },
  "http_port": 9998,
  "whitelist_count": 30,
  "command_timeout": 60,
  "timestamp": "2026-04-05T12:00:00"
}
```

#### 2. 获取白名单列表

```bash
curl http://10.10.10.200:9998/api/whitelist
```

响应：
```json
{
  "success": true,
  "whitelist": [
    "tcpdump*",
    "curl*",
    "netstat*",
    ...
  ],
  "count": 30,
  "timestamp": "2026-04-05T12:00:00"
}
```

#### 3. 添加白名单模式

```bash
curl -X POST http://10.10.10.200:9998/api/whitelist/add \
  -H "Content-Type: application/json" \
  -d '{"pattern": "iptables*"}'
```

响应：
```json
{
  "success": true,
  "message": "已添加白名单模式: iptables*",
  "whitelist": [...],
  "timestamp": "2026-04-05T12:00:00"
}
```

#### 4. 移除白名单模式

```bash
curl -X POST http://10.10.10.200:9998/api/whitelist/remove \
  -H "Content-Type: application/json" \
  -d '{"pattern": "iptables*"}'
```

#### 5. 检查命令是否允许

```bash
curl -X POST http://10.10.10.200:9998/api/command/check \
  -H "Content-Type: application/json" \
  -d '{"command": "tcpdump -i any udp port 51874"}'
```

响应：
```json
{
  "success": true,
  "command": "tcpdump -i any udp port 51874",
  "allowed": true,
  "timestamp": "2026-04-05T12:00:00"
}
```

#### 6. 执行命令（通过 HTTP）

```bash
curl -X POST http://10.10.10.200:9998/api/command/execute \
  -H "Content-Type: application/json" \
  -d '{"command": "netstat -anp | grep 51874"}'
```

响应：
```json
{
  "success": true,
  "command": "netstat -anp | grep 51874",
  "result": {
    "success": true,
    "output": "udp  0  0 0.0.0.0:51874  0.0.0.0:*  1234/zlmediakit",
    "error": "",
    "returncode": 0,
    "duration": 0.15
  },
  "timestamp": "2026-04-05T12:00:00"
}
```

### Python 调用示例

```python
import requests

# 获取白名单
response = requests.get('http://10.10.10.200:9998/api/whitelist')
print(response.json())

# 执行命令
response = requests.post(
    'http://10.10.10.200:9998/api/command/execute',
    json={'command': 'tcpdump -i any udp port 51874 -c 10'}
)
print(response.json())
```

## WebSocket 使用示例

### 基本命令执行

```
调试> tcpdump -i any udp port 51874 -nn
调试> netstat -anp | grep 51874
调试> curl http://localhost:5080/index/api/getMediaList
调试> ps aux | grep zlm
```

### 白名单管理

```
调试> /whitelist                              # 列出当前白名单
调试> /add tcpdump*                           # 添加白名单模式
调试> /remove tcpdump*                        # 移除白名单模式
调试> /add "cat /etc/zlmediakit/*"            # 添加带空格的模式（需要引号）
```

### 调试 RTP 流

#### 1. 抓取 RTP 数据包

```bash
# 抓取特定端口
调试> tcpdump -i any udp port 51874 -nn -v

# 抓取所有来自设备 IP 的包
调试> tcpdump -i any host 10.10.10.210 -nn

# 抓取并保存到文件
调试> tcpdump -i any udp port 51874 -w /tmp/rtp.pcap
```

#### 2. 检查 ZLM 状态

```bash
# 查看 ZLM 进程
调试> ps aux | grep zlmediakit

# 查看 ZLM 监听端口
调试> netstat -anp | grep zlmediakit

# 查看 ZLM 日志
调试> tail -n 50 /var/log/zlmediakit/zlmediakit.log

# 实时监控日志
调试> tail -f /var/log/zlmediakit/zlmediakit.log
```

#### 3. 测试网络连通性

```bash
# Ping 设备
调试> ping -c 4 10.10.10.210

# 测试端口可达性
调试> nc -zvu 10.10.10.200 51874

# 查看路由
调试> ip route
```

#### 4. 查询 ZLM API

```bash
# 获取流列表
调试> curl "http://localhost:5080/index/api/getMediaList?secret=YOUR_SECRET"

# 获取特定流信息
调试> curl "http://localhost:5080/index/api/getMediaInfo?secret=YOUR_SECRET&app=rtp&stream=STREAM_ID"

# 查看所有 RTP Server
调试> curl "http://localhost:5080/index/api/listRtpServer?secret=YOUR_SECRET"
```

## 白名单配置说明

### 通配符规则

| 模式 | 含义 | 示例 |
|------|------|------|
| `命令名*` | 允许该命令的所有参数组合 | `tcpdump*` → `tcpdump -i any udp port 51874` |
| `命令名 参数*` | 允许特定参数模式 | `cat /var/log/*` → `cat /var/log/zlmediakit.log` |
| `完整命令` | 只允许精确匹配 | `pwd` → 只允许 `pwd`，不允许 `pwd -P` |

### 安全建议

1. **最小权限原则**：只添加必要的命令
2. **避免危险命令**：不要添加 `rm*`、`dd*`、`mkfs*` 等
3. **限制文件访问**：使用具体的路径模式，如 `cat /var/log/*` 而不是 `cat*`
4. **网络安全**：
   - 不要监听公网 IP（使用 `127.0.0.1` 或内网 IP）
   - 考虑添加认证机制
   - 使用防火墙限制访问

### 示例配置

```python
"whitelist": [
    # 网络诊断
    "tcpdump -i any udp port *",
    "tcpdump -i any host *",
    "netstat -anp",
    "ss -anp",
    "ip *",
    "ping -c *",

    # 文件查看（限制路径）
    "cat /var/log/zlmediakit/*",
    "cat /tmp/*",
    "tail *",
    "head *",

    # 进程管理
    "ps aux",
    "ps -ef",

    # ZLM API（限制 URL）
    "curl http://localhost:5080/*",

    # 系统信息
    "uname -a",
    "hostname",
    "uptime",
    "free -h",
    "df -h",
]
```

## 常见问题

### 1. 连接失败

**症状**: 客户端无法连接到服务端

**检查**:
```bash
# 在 ZLM 服务器上
netstat -anp | grep 9999          # 检查端口是否监听
iptables -L -n                    # 检查防火墙规则
```

**解决**:
```bash
# 开放防火墙端口
sudo iptables -A INPUT -p tcp --dport 9999 -j ACCEPT

# 或修改服务端监听地址为内网 IP
```

### 2. 命令执行失败

**症状**: 命令返回 "命令不在白名单中"

**解决**:
```bash
# 方法 1: 使用客户端添加白名单
调试> /add your-command*

# 方法 2: 修改服务端配置后重启
```

### 3. 命令执行超时

**症状**: 命令返回 "命令执行超时"

**解决**:
```python
# 修改配置中的超时时间
"command_timeout": 120,  # 增加到 120 秒
```

### 4. 权限不足

**症状**: 某些命令返回权限错误

**解决**:
```bash
# 方法 1: 以 root 用户运行服务端
sudo python3 zlm_debug_server.py

# 方法 2: 使用 sudo 命令（需要添加到白名单）
# 白名单: "sudo tcpdump*", "sudo netstat*"
```

## 高级用法

### 1. 批量命令

使用客户端脚本编写自动化测试：

```python
import asyncio
from zlm_debug_client import DebugClient

async def test_rtp_flow():
    client = DebugClient("ws://10.10.10.200:9999")
    await client.connect()

    # 启动抓包
    await client.send_command("tcpdump -i any udp port 51874 -w /tmp/test.pcap &")

    # 等待流建立
    await asyncio.sleep(10)

    # 检查流状态
    await client.send_command("curl 'http://localhost:5080/index/api/getMediaList?secret=XXX'")

    # 停止抓包
    await client.send_command("pkill tcpdump")

    await client.close()

asyncio.run(test_rtp_flow())
```

### 2. 日志监控

实时监控 ZLM 日志：

```bash
调试> tail -f /var/log/zlmediakit/zlmediakit.log | grep -E "RTP|51874|SSRC"
```

### 3. 多服务器管理

修改客户端支持多个服务器：

```python
SERVERS = {
    "zlm1": "ws://10.10.10.200:9999",
    "zlm2": "ws://10.10.10.201:9999",
}
```

## 故障排除清单

### RTP 流未建立

1. ✅ 检查 RTP Server 是否创建
   ```bash
   调试> curl "http://localhost:5080/index/api/listRtpServer?secret=XXX"
   ```

2. ✅ 抓包验证 RTP 数据
   ```bash
   调试> tcpdump -i any udp port 51874 -nn
   ```

3. ✅ 检查防火墙规则
   ```bash
   调试> iptables -L -n -v
   ```

4. ✅ 测试网络连通性
   ```bash
   调试> ping -c 4 10.10.10.210
   调试> nc -zvu 10.10.10.200 51874
   ```

5. ✅ 查看 ZLM 日志
   ```bash
   调试> tail -n 100 /var/log/zlmediakit/zlmediakit.log
   ```

## 安全提醒

⚠️ **重要**：
1. 此工具仅用于调试，不要在生产环境长期运行
2. 使用完毕后立即关闭服务端
3. 不要在公网环境部署
4. 定期审计白名单配置
5. 考虑添加 TLS 加密和认证机制

## 相关文件

- `zlm_debug_server.py`: 服务端脚本（部署在 ZLM 服务器）
- `zlm_debug_client.py`: 客户端脚本（在本地运行）
- `README_DEBUG_TOOLS.md`: 本文档

## 更新日志

- 2026-04-05: 初始版本
  - WebSocket 服务器/客户端
  - 白名单机制
  - 交互式命令行