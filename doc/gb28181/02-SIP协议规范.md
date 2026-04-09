# GB/T 28181-2016 SIP协议规范

## SIP协议概述

GB/T 28181 基于 IETF RFC 3261 定义的 SIP 协议进行扩展，用于视频监控联网系统的会话控制。

## SIP网络结构

### SIP监控域结构

```
                    ┌─────────────────┐
                    │   上级SIP域      │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
        ┌─────┴─────┐  ┌─────┴─────┐  ┌─────┴─────┐
        │ SIP客户端  │  │ SIP设备   │  │ 下级SIP域 │
        └───────────┘  └───────────┘  └───────────┘
```

### 功能实体交互

SIP监控域内的功能实体包括：
1. **SIP设备**: IPC、DVR、NVR等，作为SIP UA
2. **SIP客户端**: 用户终端，作为SIP UA
3. **中心信令控制服务器**: 作为代理服务器、注册服务器
4. **媒体服务器**: 作为B2BUA处理媒体流

## SIP消息类型

### 请求消息

| 方法 | 用途 |
|------|------|
| REGISTER | 设备注册 |
| INVITE | 建立媒体会话（点播、回放） |
| ACK | 确认INVITE最终响应 |
| BYE | 结束媒体会话 |
| MESSAGE | 传输控制消息（MANSCDP） |
| INFO | 传输回放控制命令（MANSRTSP） |
| SUBSCRIBE | 订阅事件通知 |
| NOTIFY | 事件通知 |
| CANCEL | 取消请求 |
| OPTIONS | 能力查询 |

### 响应消息

| 状态码 | 说明 |
|--------|------|
| 100 | Trying - 正在处理 |
| 180 | Ringing - 振铃 |
| 200 | OK - 成功 |
| 400 | Bad Request - 请求错误 |
| 401 | Unauthorized - 未授权（需要认证） |
| 403 | Forbidden - 禁止访问 |
| 404 | Not Found - 用户不存在 |
| 408 | Request Timeout - 请求超时 |
| 486 | Busy Here - 用户忙 |
| 487 | Request Terminated - 请求终止 |
| 488 | Not Acceptable Here - 无法接受 |
| 500 | Server Internal Error - 服务器内部错误 |
| 503 | Service Unavailable - 服务不可用 |

## SIP消息头规范

### 必选头域

| 头域 | 说明 | 格式要求 |
|------|------|----------|
| Via | 消息传输路径 | 必须包含 `rport` 参数 |
| From | 消息发起方 | 格式: `<sip:ID@domain>;tag=xxx` |
| To | 消息接收方 | 格式: `<sip:ID@domain>` |
| Call-ID | 会话标识 | 全局唯一 |
| CSeq | 命令序列号 | 整数 + 方法名 |
| Max-Forwards | 最大转发次数 | 通常为70 |
| Contact | 联系地址 | 格式: `<sip:IP:Port>` |
| Date | 日期时间 | 注册响应必须包含，格式: `yyyy-MM-dd'T'HH:mm:ss.SSS` |

### 重要头域规则

#### From/To 头域

- **禁止使用 IP:Port 格式**，必须使用20位设备ID作为用户名
- 必须包含 `tag` 参数（From必须，To在响应中必须）

```
From: <sip:34020000002000000001@3402000000>;tag=1928301774
To: <sip:34020000001320000001@3402000000>
```

#### Via 头域

- 必须包含 `rport` 参数用于NAT场景
- 每经过一个代理增加一个Via

```
Via: SIP/2.0/UDP 192.168.1.100:5060;rport;branch=z9hG4bK-xxx
```

#### Contact 头域

- 包含设备的实际IP和端口
- 用于后续请求的路由

```
Contact: <sip:34020000002000000001@192.168.1.100:5060>
```

## SIP URI 格式

### 标准格式

```
sip[s]:username@domain[:port][;uri-parameters]
```

### GB28181规范

- **username**: 20位设备ID或18位域编码
- **domain**: 域ID（20位编码）或 `.spvmn.cn`
- **port**: 默认5060，可省略

### 示例

```
sip:34020000002000000001@3402000000
sip:34020000002000000001@192.168.1.100:5060
sips:34020000002000000001@3402000000
```

## SIP认证机制

### Digest认证流程

```
设备                                    服务器
  │                                       │
  │ ──────── REGISTER ──────────────────> │
  │                                       │
  │ <─────── 401 Unauthorized ─────────── │
  │          WWW-Authenticate             │
  │                                       │
  │ ──────── REGISTER ──────────────────> │
  │          Authorization                │
  │                                       │
  │ <─────── 200 OK ──────────────────── │
```

### WWW-Authenticate 头域

```
WWW-Authenticate: Digest realm="3402000000", nonce="xxx", algorithm=MD5
```

### Authorization 头域

```
Authorization: Digest username="34020000002000000001",
               realm="3402000000",
               nonce="xxx",
               uri="sip:3402000000@192.168.1.100:5060",
               response="xxx",
               algorithm=MD5
```

### response 计算公式

```
response = MD5(HA1:nonce:HA2)
```

**计算步骤**:
1. HA1 = MD5(username:realm:password)
2. HA2 = MD5(method:uri)
3. response = MD5(HA1:nonce:HA2)

**示例计算**:
```
假设:
  username = "34020000002000000001"
  realm = "3402000000"
  password = "12345678"
  nonce = "6fe9ba44a76be22a"
  method = "REGISTER"
  uri = "sip:3402000000@192.168.1.100:5060"

计算:
  HA1 = MD5("34020000002000000001:3402000000:12345678")
      = "2a3b4c5d6e7f8g9h..."

  HA2 = MD5("REGISTER:sip:3402000000@192.168.1.100:5060")
      = "1a2b3c4d5e6f7g8h..."

  response = MD5("HA1:nonce:HA2")
           = "9625d92d1bddea7a911926e0db054968"
```

### 数字摘要认证（附录H）

支持非对称加密认证：
- 支持 RSA 加密
- 支持 SHA-1、SHA-256 哈希算法

## Subject 头域规范

Subject 头域用于传递媒体会话相关参数。

### 格式

```
Subject: 通道ID:SSRC,平台ID:0
```

### 参数说明

| 参数 | 说明 |
|------|------|
| 通道ID | 20位设备/通道编码 |
| SSRC | 10位SSRC值，用于标识媒体流 |
| 平台ID | 20位平台编码 |
| 第二通道ID | 通常为0 |

### 示例

```
Subject: 34020000001310000001:0501007226,44050100002000000002:0
```

### 说明

- 逗号分隔发送端和接收端信息
- SSRC 由平台生成，用于媒体流标识
- 不同场景（点播/回放/下载）通过 SDP 的 `s` 字段区分，不通过 Subject 区分

## Allow 头域

服务器和设备应支持以下方法：

```
Allow: INVITE, ACK, INFO, CANCEL, BYE, OPTIONS, MESSAGE
```

**注意**: 对于事件订阅场景，还需支持 SUBSCRIBE 和 NOTIFY 方法。

## Expires 头域

Expires 头域用于指定注册有效期。

### 注册场景

| 场景 | Expires值 | 说明 |
|------|-----------|------|
| 注册 | 3600-86400 | 有效期，建议3600秒 |
| 注销 | 0 | 立即注销 |
| 刷新 | 当前有效期 | 在到期前刷新 |

### 时间要求

- **最大有效期**: 86400秒（1天）
- **建议有效期**: 3600秒（1小时）
- **刷新时机**: 有效期过半时刷新
- **心跳周期**: 建议为有效期的1/3

### 示例

```
REGISTER sip:3402000000@192.168.1.100:5060 SIP/2.0
Expires: 3600
```

### 注销示例

```
REGISTER sip:3402000000@192.168.1.100:5060 SIP/2.0
Expires: 0
```

## Content-Type 头域

| 内容类型 | 用途 |
|----------|------|
| application/sdp | SDP会话描述 |
| Application/MANSCDP+xml | 设备控制、查询消息 |
| Application/MANSRTSP | 录像回放控制 |

## 传输协议

### UDP 传输

- 默认传输方式
- 端口：5060
- 适用于小消息

### TCP 传输

- 支持NAT穿透
- 支持大消息传输
- 用于媒体流传输（附录L）

### TLS 传输

- 加密传输
- 用于安全通信

## NAT 穿越支持

1. Via 头域包含 `rport` 参数
2. 响应消息发送到请求源地址
3. 支持 TCP 传输方式

## 典型消息示例

### REGISTER 请求

```sip
REGISTER sip:3402000000@192.168.1.100:5060 SIP/2.0
Via: SIP/2.0/UDP 192.168.1.200:5060;rport;branch=z9hG4bK-xxx
From: <sip:34020000002000000001@3402000000>;tag=xxx
To: <sip:3402000000@3402000000>
Call-ID: xxx@192.168.1.200
CSeq: 1 REGISTER
Contact: <sip:34020000002000000001@192.168.1.200:5060>
Max-Forwards: 70
Expires: 3600
Content-Length: 0
```

### INVITE 请求

```sip
INVITE sip:34020000001320000001@192.168.1.200:5060 SIP/2.0
Via: SIP/2.0/UDP 192.168.1.100:5060;rport;branch=z9hG4bK-xxx
From: <sip:34020000002000000001@3402000000>;tag=xxx
To: <sip:34020000001320000001@3402000000>
Call-ID: xxx@192.168.1.100
CSeq: 1 INVITE
Contact: <sip:192.168.1.100:5060>
Subject: 34020000001320000001:34020000001320000001,34020000002000000001:0
Content-Type: application/sdp
Max-Forwards: 70
Content-Length: xxx

[SDP内容]
```

### MESSAGE 请求

```sip
MESSAGE sip:34020000001320000001@192.168.1.200:5060 SIP/2.0
Via: SIP/2.0/UDP 192.168.1.100:5060;rport;branch=z9hG4bK-xxx
From: <sip:34020000002000000001@3402000000>;tag=xxx
To: <sip:34020000001320000001@3402000000>
Call-ID: xxx@192.168.1.100
CSeq: 1 MESSAGE
Content-Type: Application/MANSCDP+xml
Max-Forwards: 70
Content-Length: xxx

<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <PTZCmd>A50F4D1000001021</PTZCmd>
</Control>
```

## 注意事项

1. **ACK 不应终止 INVITE 事务**：在 INVITE 事务中使用 `defer tx.Terminate()` 会导致事务提前终止
2. **From 头域必须使用域编码**：禁止使用 IP:Port 格式
3. **Via 必须包含 rport**：用于 NAT 场景
4. **ACK Request-URI 必须与 INVITE 相同**：不能修改
5. **Subject 头域格式固定**：必须按照规范格式填写