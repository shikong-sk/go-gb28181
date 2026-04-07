# go-gb28181

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Vue](https://img.shields.io/badge/Vue-3.4+-4FC08D?style=flat&logo=vue.js)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

GB/T 28181-2016 国标视频监控协议的 Go 语言实现。

## ✨ 功能特性

### 设备端 (Client)
- ✅ SIP 注册与 Digest 认证
- ✅ 心跳保活 (30s 间隔)
- ✅ 目录查询响应 (Catalog)
- ✅ 报警通知上报
- ✅ 配置化设备信息 (名称、厂商、型号)

### 服务端 (Server)
- ✅ 设备注册管理
- ✅ Digest 认证验证 (支持设备独立密码)
- ✅ 目录同步与订阅 (CatalogSubscription)
- ✅ 视频点播 (INVITE/ACK/BYE)
  - ✅ 实时视频播放
  - ✅ 多观众流共享 (延迟关流机制)
  - ✅ ZLMediaKit Hook 集成
- ✅ 历史录像查询与回放
- ✅ 云台控制 (PTZ)
  - ✅ Pelco-D 扩展格式 (8字节命令)
  - ✅ 方向控制 (上/下/左/右/停止)
  - ✅ Zoom 控制 (放大/缩小)
  - ✅ 控制优先级支持
- ✅ 报警记录持久化
- ✅ HTTP RESTful API
- ✅ Vue3 前端管理界面
- ✅ 多网卡场景支持
- ✅ 端口冲突检测 (HTTP/SIP)

---

## 📋 后续计划

- [ ] 设备状态实时更新 (WebSocket/SSE)
- [ ] 移动设备定位 (MobilePosition)
- [ ] 设备状态查询 (DeviceStatus)
- [ ] 录像下载功能

---

## 🚀 快速开始

### 环境要求

- Go 1.25+
- Node.js 18+ (前端开发)
- SQLite / MySQL / PostgreSQL
- ZLMediaKit (流媒体服务)

### 安装

```bash
# 克隆项目
git clone https://github.com/shikong/go-gb28181.git
cd go-gb28181

# 下载依赖
go mod download

# 编译服务端
go build -o bin/server ./cmd/server

# 编译设备端
go build -o bin/client ./cmd/client

# 编译前端
cd frontend && npm install && npm run build
```

### 运行

```bash
# 运行服务端
./bin/server -config config.toml

# 运行设备端
./bin/client -config client.toml

# 开发模式 (前端)
cd frontend && npm run dev
```

---

## 📁 项目结构

```
go-gb28181/
├── cmd/                    # 入口点
│   ├── client/             # 设备端入口
│   └── server/             # 服务端入口
├── internal/               # 内部代码
│   ├── client/             # 设备端实现
│   │   ├── config/         # 配置
│   │   ├── handler/        # SIP 消息处理
│   │   └── services/       # 业务服务
│   └── server/             # 服务端实现
│       ├── api/            # HTTP API
│       ├── app/            # 应用管理
│       ├── config/         # 配置
│       ├── database/       # 数据库
│       ├── model/          # 数据模型
│       ├── repository/     # 数据访问层
│       ├── service/        # 业务服务层
│       └── sip/            # SIP 服务
├── pkg/                    # 共享代码
│   ├── manscdp/            # MANSCDP 协议
│   ├── services/           # 外部服务
│   ├── utils/              # 工具函数
│   └── log/                # 日志
└── frontend/               # Vue3 前端
```

---

## ⚙️ 配置说明

### 服务端配置 (config.toml)

```toml
# 调试模式
debug = true

[sip]
# SIP 是核心功能，必须配置
# SIP 服务器配置
server_id = "34020000002000000001"   # 服务器 ID (20位国标编码)
server_ip = "10.10.10.30"            # 服务器 IP
server_port = 5060                   # 服务器端口

# 本地配置
device_id = "44050100002000000002"   # 设备 ID (20位国标编码)
# 监听 IP：设备访问的网卡 IP
# 多网卡主机需指定具体 IP（如内网网卡 IP），让设备能访问到
# 0.0.0.0 表示监听所有网卡
listen_ip = "10.10.10.30"
listen_port = 5099                   # 监听端口

# 对外 IP：SIP 消息中的 IP（设备回连时使用）
# 为空时自动检测本机 IP，多网卡时建议手动指定
external_ip = ""

# 认证配置
password = "12345678"                # 密码

# 注册周期(秒)
register_cycle = 3600

# INVITE 超时(秒)
invite_timeout = 15

[http]
# HTTP API 配置（用户访问）
# 多网卡主机建议指定用户可访问的网卡 IP
enabled = true
host = "0.0.0.0"                     # 监听地址（用户访问的网卡 IP）
port = 8080                          # 监听端口
log_file = "./logs/server.log"       # 日志文件路径（为空则仅输出到控制台）
daemonize = false                    # 是否自动切换为后台运行模式

[database]
type = "sqlite"                      # 数据库类型: sqlite, mysql, postgres
name = "data/gb28181.db"             # 数据库文件路径 (sqlite) 或数据库名 (mysql/postgres)
max_open_conns = 10                  # 最大打开连接数
max_idle_conns = 5                   # 最大空闲连接数

[zlmediakit]
url = "http://10.10.10.200:5080"     # ZLMediaKit 地址
secret = "4155cca6-2f9f-11ee-85e6-8de4ce2e7333"  # API 密钥
id = "amrWMKmbKqoBjRQ9"              # ZLMediaKit ID
rtp_port = 0                         # RTP 端口 (0=自动分配)

[alarm]
enabled = true                       # 是否保存报警记录
retention_days = 3                   # 保留天数 (0=永久保留)

[redis]
addr = "10.10.10.200:16379"          # Redis 地址 (用于 SSRC 池管理)
password = "12341234"                # Redis 密码
db = 6                               # Redis 数据库编号
```

> **多网卡场景**: 当服务器有多个网卡时，SIP 监听 IP (`listen_ip`) 应设置为设备可访问的内网 IP，
> HTTP 监听 IP (`host`) 可设置为 `0.0.0.0` 允许所有用户访问。ZLMediaKit Hook 回调 URL 会自动处理 `0.0.0.0` 情况。

### 设备端配置 (client.toml)

```toml
debug = false

serverIp = "10.10.10.20"              # 平台IP
serverPort = 5060                     # 平台端口
serverId = "44050100002000000003"     # 平台ID
password = "123456"                   # 认证密码

deviceId = "44050100002000000002"     # 设备ID
name = "GB28181设备"                  # 设备名称
manufacturer = "Shikong"              # 设备厂商
model = "SK-GB28181-001"             # 设备型号
ipAddress = "10.10.10.100"           # 设备IP
listenIp = "0.0.0.0"
listenPort = 5099
```

---

## 📡 API 文档

### 设备管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 设备列表 |
| GET | `/api/devices/:id` | 设备详情 |
| POST | `/api/devices/:id/sync` | 同步设备目录 |
| DELETE | `/api/devices/:id` | 删除设备 |

### 通道管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/channels` | 通道列表 |
| GET | `/api/channels/:id` | 通道详情 |

### 视频播放

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/play/start` | 开始播放 |
| POST | `/api/play/stop` | 停止播放 |
| GET | `/api/play/sessions` | 播放会话列表 |
| GET | `/api/play/media/:stream_id` | 获取流信息 |

### 云台控制 (PTZ)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/ptz/control` | 云台控制 |
| POST | `/api/ptz/stop` | 停止云台 |

**云台控制请求示例**:
```json
{
  "device_id": "44050100001320000001",
  "channel_id": "44050100001310000001",
  "direction": "Left",
  "horizontal_speed": 30,
  "vertical_speed": 30
}
```

**支持的方向**:
- `Stop` - 停止
- `Up` / `Down` - 上/下
- `Left` / `Right` - 左/右
- `UpLeft` / `UpRight` - 上左/上右
- `DownLeft` / `DownRight` - 下左/下右
- `ZoomIn` / `ZoomOut` - 放大/缩小

### 报警管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alarms` | 报警列表 (支持筛选) |
| GET | `/api/alarms/config` | 报警配置 |
| GET | `/api/alarms/:id` | 报警详情 |
| DELETE | `/api/alarms/:id` | 删除报警 |
| POST | `/api/alarms/subscribe` | 全局报警订阅 |

### 订阅管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/subscriptions` | 订阅列表 |
| POST | `/api/devices/:device_id/subscribe/catalog` | 订阅设备目录 |
| DELETE | `/api/devices/:device_id/subscribe/catalog` | 取消目录订阅 |
| POST | `/api/devices/:device_id/subscribe/alarm` | 订阅设备报警 |
| DELETE | `/api/devices/:device_id/subscribe/alarm` | 取消报警订阅 |

### 历史录像

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/record/list` | 录像文件列表 |
| POST | `/api/record/fetch` | 请求设备上传录像 |
| GET | `/api/record/fetch/status` | 获取上传状态 |

### 录像下载

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/download/start` | 开始下载 |
| GET | `/api/download/progress/:stream_id` | 下载进度 |
| POST | `/api/download/cancel/:stream_id` | 取消下载 |
| GET | `/api/download/sessions` | 下载会话列表 |

### 设备状态与信息

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices/:device_id/status` | 设备状态 |
| POST | `/api/devices/:device_id/query-status` | 查询设备状态 |
| POST | `/api/devices/:device_id/query-info` | 查询设备信息 |
| GET | `/api/devices/:device_id/info` | 设备信息 |
| GET | `/api/devices/:device_id/position` | 最新位置 |
| GET | `/api/devices/:device_id/positions` | 位置历史 |

### WebSocket

| 路径 | 说明 |
|------|------|
| `/ws/events` | 实时事件推送 (设备状态/报警等) |

---

## 🔧 开发指南

### 架构模式

```
Model → Repository → Service → Handler/API
```

### 关键约定

1. **XML 解码**: 使用 `utils.XMLUnmarshal` 自动处理 GBK/GB2312/UTF-8
2. **日志**: 使用 `pkg/log` (zerolog 封装)
3. **数据库**: GORM，模型需实现 `TableName()` 方法

### SIP 消息流

```
设备 -> 平台: REGISTER
平台 -> 设备: 401 Unauthorized
设备 -> 平台: REGISTER + Digest认证
平台 -> 设备: 200 OK
设备 -> 平台: MESSAGE (Keepalive/30s)
设备 -> 平台: MESSAGE (Catalog响应)
平台 -> 设备: MESSAGE (DeviceControl/云台控制)
设备 -> 平台: 200 OK
```

### PTZ 云台控制命令格式

采用 GB28181 附录 D 规定的 Pelco-D 扩展格式 (8字节):

```
字节0-1: 同步头 (A5 0F 固定)
字节2: 组合码 (云台控制固定 01)
字节3: 命令码 (方向控制位)
字节4: 水平速度 (0-255)
字节5: 垂直速度 (0-255)
字节6: Zoom速度/预留
字节7: 校验和 (前7字节累加 mod 256)

命令码位定义:
- Bit0 (0x01): 右转
- Bit1 (0x02): 左转
- Bit2 (0x04): 下转
- Bit3 (0x08): 上转
- Bit4 (0x10): 放大
- Bit5 (0x20): 缩小

示例:
- 左转(速度30): A50F01021E1E1003
- 停止: A50F0100000000B5
```

### 运行测试

```bash
go test ./...
```

---

## 🛠️ 技术栈

### 后端
- **Go 1.25** - 核心语言
- **sipgo** - SIP 协议栈
- **Gin** - HTTP 框架
- **GORM** - ORM
- **zerolog** - 日志
- **icholy/digest** - Digest 认证

### 前端
- **Vue 3.4** - 框架
- **TypeScript 5.3** - 类型安全
- **Element Plus** - UI 组件
- **Pinia** - 状态管理
- **Tailwind CSS** - 样式
- **Vite** - 构建工具

---

## 📄 License

[MIT License](LICENSE)

---

## 🙏 致谢

- [sipgo](https://github.com/emiago/sipgo) - Go SIP 协议栈
- [ZLMediaKit](https://github.com/ZLMediaKit/ZLMediaKit) - 流媒体服务器
