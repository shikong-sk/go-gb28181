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
- ✅ 目录同步
- ✅ 视频点播 (INVITE/ACK/BYE)
- ✅ 历史录像查询与回放
- ✅ 云台控制
- ✅ 报警记录持久化
- ✅ HTTP RESTful API
- ✅ Vue3 前端管理界面

---

## 📋 后续计划

- [x] ~~视频点播完善~~
  - ✅ PlayService 注入 DeviceService 获取设备真实 IP
  - ✅ INVITE 发送到设备真实地址
  - ✅ BYE 使用正确的 channelId + 设备 IP
  - ✅ BYE 使用 session 保存的 Dialog 信息 (Via/Call-ID/From/To)
- [x] ~~历史录像回放~~ (已实现 PlayBack 方法)
- [x] ~~云台控制~~ (已实现 PTZService)
- [ ] 设备状态实时更新 (WebSocket/SSE)
- [ ] 移动设备定位 (MobilePosition)
- [ ] 设备状态查询 (DeviceStatus)

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
enabled = true
server_id = "34020000001320000001"   # 服务器ID (20位国标编码)
server_ip = "10.10.10.95"             # 服务器IP
server_port = 5060                    # SIP端口
password = "12345678"                 # 认证密码

[http]
enabled = true
host = "0.0.0.0"
port = 8080

[database]
type = "sqlite"                       # sqlite / mysql / postgres
name = "data/gb28181.db"

[zlmediakit]
url = "http://127.0.0.1:80"           # ZLMediaKit地址
secret = ""

[alarm]
enabled = true                        # 是否保存报警记录
retention_days = 3                    # 保留天数 (0=永久保留)
```

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

### 报警管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/alarms` | 报警列表 (支持筛选) |
| GET | `/api/alarms/config` | 报警配置 |
| GET | `/api/alarms/:id` | 报警详情 |
| DELETE | `/api/alarms/:id` | 删除报警 |
| DELETE | `/api/alarms` | 清空所有报警 |

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
