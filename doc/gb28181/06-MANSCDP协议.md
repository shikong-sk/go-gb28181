# GB/T 28181-2016 MANSCDP协议

## 协议概述

MANSCDP (Monitoring and Alarming Network System Control Description Protocol) 是视频监控联网系统的控制描述协议，用于设备控制、信息查询、事件通知等业务。

## 消息类型

| 消息类型 | 说明 | 用途 |
|----------|------|------|
| Control | 控制命令 | 设备控制（云台、录像等） |
| Query | 查询命令 | 设备信息、目录、状态查询 |
| Notify | 通知消息 | 心跳、报警、状态变化通知 |
| Response | 响应消息 | 对Control/Query的响应 |

### 命令类型 (CmdType)

| CmdType | 说明 | 消息类型 |
|---------|------|----------|
| Keepalive | 心跳保活 | Notify |
| Catalog | 目录查询/通知 | Query/Notify/Response |
| DeviceInfo | 设备信息查询 | Query/Response |
| DeviceStatus | 设备状态查询 | Query/Response |
| DeviceControl | 设备控制（含PTZ/录像/拉框/看守位等） | Control/Response |
| DeviceConfig | 设备配置 | Control/Response |
| RecordInfo | 录像文件查询 | Query/Response |
| Alarm | 报警查询/通知 | Query/Notify |
| MediaStatus | 媒体状态通知 | Notify |
| ConfigDownload | 配置下载 | Query/Response |
| PresetQuery | 预置位查询 | Query/Response |
| MobilePosition | 移动设备位置查询/通知 | Query/Notify |
| Broadcast | 语音广播通知/响应 | Notify/Response |

**注意**: DragZoomIn、DragZoomOut、HomePosition、TeleBoot、RecordCmd 是 DeviceControl 的子元素，不是独立的 CmdType。

## XML消息格式

### 基本结构

```xml
<?xml version="1.0"?>
<命令类型>
  <CmdType>命令代码</CmdType>
  <SN>序列号</SN>
  <DeviceID>设备ID</DeviceID>
  <!-- 其他参数 -->
</命令类型>
```

### 公共字段

| 字段 | 类型 | 说明 |
|------|------|------|
| CmdType | string | 命令类型代码 |
| SN | integer | 序列号（1-65535） |
| DeviceID | string | 设备ID（20位） |

## Control 控制命令

### 设备控制

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>11</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <PTZCmd>A50F4D1000001021</PTZCmd>
  <Info>
    <ControlPriority>5</ControlPriority>
  </Info>
</Control>
```

### 云台控制命令 (PTZCmd)

PTZCmd为8字节十六进制字符串：

| 字节 | 说明 |
|------|------|
| 1 | A5H（起始码，固定） |
| 2 | 0FH（同步字节，固定） |
| 3 | 01H（固定） |
| 4 | 指令码（方向+动作） |
| 5 | 水平速度（00H-FFH） |
| 6 | 垂直速度（00H-FFH） |
| 7 | 组合码（预置位/聚焦/光圈） |
| 8 | 校验码（字节1-7累加和 mod 256） |

**指令码（字节4）编码**:

| 指令码 | 动作 |
|--------|------|
| 00H | 停止 |
| 01H | 左转 |
| 02H | 右转 |
| 04H | 上转 |
| 08H | 下转 |
| 10H | 变倍近 |
| 20H | 变倍远 |

**组合控制**：多个位组合实现，如左上=01H+04H=05H

**校验码计算**：`(Byte1 + Byte2 + Byte3 + Byte4 + Byte5 + Byte6 + Byte7) mod 256`

### 录像控制

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>17</SN>
  <DeviceID>34020000002000000001</DeviceID>
  <RecordCmd>Record</RecordCmd>
</Control>
```

**RecordCmd值**:
| 值 | 说明 |
|----|------|
| Record | 开始录像 |
| StopRecord | 停止录像 |

### 拉框放大控制

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <DragZoomIn>
    <TopLeftX>100</TopLeftX>
    <TopLeftY>100</TopLeftY>
    <BottomRightX>200</BottomRightX>
    <BottomRightY>200</BottomRightY>
  </DragZoomIn>
</Control>
```

### 拉框缩小控制

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <DragZoomOut>
    <TopLeftX>100</TopLeftX>
    <TopLeftY>100</TopLeftY>
    <BottomRightX>200</BottomRightX>
    <BottomRightY>200</BottomRightY>
  </DragZoomOut>
</Control>
```

### 看守位控制

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <HomePosition>
    <Enabled>1</Enabled>
    <PresetIndex>1</PresetIndex>
    <ResetTime>10</ResetTime>
  </HomePosition>
</Control>
```

**HomePosition字段**:
| 字段 | 说明 |
|------|------|
| Enabled | 是否启用（0/1） |
| PresetIndex | 预置位号 |
| ResetTime | 返回时间（秒） |

### 远程启动控制

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceControl</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <TeleBoot>Boot</TeleBoot>
</Control>
```

### 设备配置

```xml
<?xml version="1.0"?>
<Control>
  <CmdType>DeviceConfig</CmdType>
  <SN>1</SN>
  <DeviceID>34020000002000000001</DeviceID>
  <BasicParam>
    <Name>设备名称</Name>
    <Expiration>3600</Expiration>
    <HeartBeatInterval>60</HeartBeatInterval>
    <HeartBeatCount>3</HeartBeatCount>
  </BasicParam>
</Control>
```

## Query 查询命令

### 目录查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>Catalog</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001110000001</DeviceID>
</Query>
```

### 设备信息查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>DeviceInfo</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001110000001</DeviceID>
</Query>
```

### 设备状态查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>DeviceStatus</CmdType>
  <SN>248</SN>
  <DeviceID>34020000001110000001</DeviceID>
</Query>
```

### 录像文件查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>RecordInfo</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001310000001</DeviceID>
  <StartTime>2010-11-11T19:46:17</StartTime>
  <EndTime>2010-11-12T19:46:17</EndTime>
  <FilePath>34020000002100000001</FilePath>
  <Secrecy>0</Secrecy>
  <Type>time</Type>
</Query>
```

### 报警查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>Alarm</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001340000001</DeviceID>
  <StartAlarmPriority>1</StartAlarmPriority>
  <EndAlarmPriority>4</EndAlarmPriority>
  <AlarmMethod>0</AlarmMethod>
  <StartTime>2010-11-11T00:00:00</StartTime>
  <EndTime>2010-12-11T00:00:00</EndTime>
</Query>
```

### 配置下载

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>ConfigDownload</CmdType>
  <SN>1</SN>
  <DeviceID>34020000002000000001</DeviceID>
  <ConfigType>BasicParam</ConfigType>
</Query>
```

### 预置位查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>PresetQuery</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001320000001</DeviceID>
</Query>
```

### 移动设备位置查询

```xml
<?xml version="1.0"?>
<Query>
  <CmdType>MobilePosition</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001310000001</DeviceID>
  <Interval>5</Interval>
</Query>
```

**参数说明**:
| 字段 | 说明 |
|------|------|
| Interval | 位置上报间隔（秒） |

## Notify 通知消息

### 心跳通知

```xml
<?xml version="1.0"?>
<Notify>
  <CmdType>Keepalive</CmdType>
  <SN>43</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <Status>OK</Status>
</Notify>
```

### 报警通知

```xml
<?xml version="1.0"?>
<Notify>
  <CmdType>Alarm</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001340000101</DeviceID>
  <AlarmPriority>4</AlarmPriority>
  <AlarmTime>2009-12-04T16:23:32</AlarmTime>
  <AlarmMethod>2</AlarmMethod>
  <Longitude>171.3</Longitude>
  <Latitude>34.2</Latitude>
</Notify>
```

**报警方法 (AlarmMethod)**:

| 值 | 说明 |
|----|------|
| 1 | 视频报警 |
| 2 | 设备故障报警 |
| 3 | GPS报警 |
| 4 | 视频信号丢失 |
| 5 | 移动侦测 |
| 6 | 设备防拆报警 |
| 7 | 其他报警 |

**报警优先级 (AlarmPriority)**:

| 值 | 说明 |
|----|------|
| 1 | 一级（高） |
| 2 | 二级 |
| 3 | 三级 |
| 4 | 四级（低） |

### 媒体状态通知

```xml
<?xml version="1.0"?>
<Notify>
  <CmdType>MediaStatus</CmdType>
  <SN>8</SN>
  <DeviceID>34020000001310000001</DeviceID>
  <NotifyType>121</NotifyType>
</Notify>
```

**NotifyType**:
- 121: 录像回放结束

### 目录订阅通知

```xml
<?xml version="1.0"?>
<Notify>
  <CmdType>Catalog</CmdType>
  <SN>162</SN>
  <DeviceID>34020000002000000001</DeviceID>
  <SumNum>2</SumNum>
  <DeviceList Num="2">
    <Item>
      <DeviceID>34020000001320000004</DeviceID>
      <Event>OFF</Event>
    </Item>
    <Item>
      <DeviceID>34020000001320000005</DeviceID>
      <Event>ON</Event>
    </Item>
  </DeviceList>
</Notify>
```

**Event状态**:

| 值 | 说明 |
|----|------|
| ON | 上线 |
| OFF | 下线 |
| VLOST | 视频丢失 |
| DEFECT | 故障 |
| ADD | 新增 |
| DEL | 删除 |
| UPDATE | 更新 |

### 移动设备位置通知

```xml
<?xml version="1.0"?>
<Notify>
  <CmdType>MobilePosition</CmdType>
  <SN>1</SN>
  <DeviceID>34020000001310000001</DeviceID>
  <Time>2010-11-11T19:46:17</Time>
  <Longitude>171.3</Longitude>
  <Latitude>34.2</Latitude>
  <Speed>60</Speed>
  <Direction>90</Direction>
  <Altitude>100</Altitude>
</Notify>
```

**位置参数**:
| 字段 | 说明 |
|------|------|
| Longitude | 经度 |
| Latitude | 纬度 |
| Speed | 速度（km/h） |
| Direction | 方向（0-360度） |
| Altitude | 海拔（米） |

### 语音广播通知

```xml
<?xml version="1.0"?>
<Notify>
  <CmdType>Broadcast</CmdType>
  <SN>992</SN>
  <SourceID>34020000001360000001</SourceID>
  <TargetID>34020000001320000001</TargetID>
</Notify>
```

### 语音广播响应

```xml
<?xml version="1.0"?>
<Response>
  <CmdType>Broadcast</CmdType>
  <SN>992</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <Result>OK</Result>
</Response>
```

## Response 响应消息

### 设备控制响应

```xml
<?xml version="1.0"?>
<Response>
  <CmdType>DeviceControl</CmdType>
  <SN>11</SN>
  <DeviceID>34020000001320000001</DeviceID>
  <Result>OK</Result>
</Response>
```

### 目录查询响应

```xml
<?xml version="1.0"?>
<Response>
  <CmdType>Catalog</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <SumNum>100</SumNum>
  <DeviceList Num="2">
    <Item>
      <DeviceID>34020000001330000001</DeviceID>
      <Name>Camera1</Name>
      <Manufacturer>Manufacturer1</Manufacturer>
      <Model>Model1</Model>
      <Owner>Owner1</Owner>
      <CivilCode>340200</CivilCode>
      <Address>Address1</Address>
      <Parental>1</Parental>
      <ParentID>34020000001110000001</ParentID>
      <SafetyWay>0</SafetyWay>
      <RegisterWay>1</RegisterWay>
      <CertNum>CertNum1</CertNum>
      <Certifiable>0</Certifiable>
      <ErrCode>400</ErrCode>
      <EndTime>2010-11-11T19:46:17</EndTime>
      <Secrecy>0</Secrecy>
      <IPAddress>192.168.3.81</IPAddress>
      <Port>5060</Port>
      <Password>Password1</Password>
      <Status>ON</Status>
      <Longitude>171.3</Longitude>
      <Latitude>34.2</Latitude>
    </Item>
  </DeviceList>
</Response>
```

### 设备信息响应

```xml
<?xml version="1.0"?>
<Response>
  <CmdType>DeviceInfo</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001110000001</DeviceID>
  <Result>OK</Result>
  <Manufacturer>Tiandy</Manufacturer>
  <Model>TC-2808AN-HD</Model>
  <Firmware>V2.1,build091111</Firmware>
</Response>
```

### 设备状态响应

```xml
<?xml version="1.0"?>
<Response>
  <CmdType>DeviceStatus</CmdType>
  <SN>248</SN>
  <DeviceID>34020000001130000001</DeviceID>
  <Result>OK</Result>
  <Online>ONLINE</Online>
  <Status>OK</Status>
  <Encode>ON</Encode>
  <Record>OFF</Record>
  <DeviceTime>2010-11-11T19:46:17</DeviceTime>
  <Alarmstatus Num="2">
    <Item>
      <DeviceID>34020000001340000001</DeviceID>
      <DutyStatus>OFFDUTY</DutyStatus>
    </Item>
  </Alarmstatus>
</Response>
```

### 录像文件响应

```xml
<?xml version="1.0"?>
<Response>
  <CmdType>RecordInfo</CmdType>
  <SN>17430</SN>
  <DeviceID>34020000001310000001</DeviceID>
  <Name>Camera1</Name>
  <SumNum>100</SumNum>
  <RecordList Num="2">
    <Item>
      <DeviceID>34020000001310000001</DeviceID>
      <Name>Camera1</Name>
      <FilePath>34020000002100000001</FilePath>
      <Address>Address1</Address>
      <StartTime>2010-11-12T10:10:00</StartTime>
      <EndTime>2010-11-12T10:20:00</EndTime>
      <Secrecy>0</Secrecy>
      <Type>time</Type>
      <RecorderID>34020000003000000001</RecorderID>
    </Item>
  </RecordList>
</Response>
```

## Item 结构说明

### 目录项 (itemType)

| 字段 | 类型 | 说明 |
|------|------|------|
| DeviceID | string | 设备ID |
| Name | string | 设备名称 |
| Manufacturer | string | 厂商 |
| Model | string | 型号 |
| Owner | string | 所属者 |
| CivilCode | string | 行政区划 |
| Block | string | 区块 |
| Address | string | 地址 |
| Parental | integer | 是否有子设备 |
| ParentID | string | 父设备ID |
| SafetyWay | integer | 安全传输方式 |
| RegisterWay | integer | 注册方式 |
| CertNum | string | 证书编号 |
| Certifiable | integer | 是否可认证 |
| ErrCode | integer | 错误码 |
| EndTime | dateTime | 结束时间 |
| Secrecy | integer | 保密级别 |
| IPAddress | string | IP地址 |
| Port | integer | 端口 |
| Password | string | 密码 |
| Status | string | 状态（ON/OFF） |
| Longitude | double | 经度 |
| Latitude | double | 纬度 |

### 录像文件项 (itemFileType)

| 字段 | 类型 | 说明 |
|------|------|------|
| DeviceID | string | 设备ID |
| Name | string | 文件名 |
| FilePath | string | 文件路径 |
| Address | string | 地址 |
| StartTime | dateTime | 开始时间 |
| EndTime | dateTime | 结束时间 |
| Secrecy | integer | 保密级别 |
| Type | string | 类型（time/alarm/manual） |
| RecorderID | string | 录像设备ID |
| FileSize | string | 文件大小 |
| RecordID | integer | 录像序号 |

### 报警信息项 (AlarmInfoType)

| 字段 | 类型 | 说明 |
|------|------|------|
| DeviceID | string | 报警设备ID |
| AlarmPriority | integer | 报警级别（1-4） |
| AlarmMethod | integer | 报警类型 |
| AlarmTime | dateTime | 报警时间 |
| AlarmDescription | string | 报警描述 |
| Longitude | double | 经度 |
| Latitude | double | 纬度 |

### 设备状态扩展字段

| 字段 | 类型 | 说明 |
|------|------|------|
| Online | string | 在线状态（ONLINE/OFFLINE） |
| Status | string | 工作状态（OK/ERROR） |
| Encode | string | 编码状态（ON/OFF） |
| Record | string | 录像状态（ON/OFF） |
| DeviceTime | dateTime | 设备时间 |
| DiskNum | integer | 硬盘数量 |
| DiskStatus | string | 硬盘状态 |

## 传输方式

MANSCDP消息通过SIP MESSAGE方法传输：

```
MESSAGE sip:设备ID@地址 SIP/2.0
Content-Type: Application/MANSCDP+xml

[XML消息体]
```

## 注意事项

1. **编码问题**
   - XML消息使用UTF-8编码
   - 中文内容需要正确处理编码

2. **SN管理**
   - SN范围：1-65535
   - 循环使用

3. **响应要求**
   - 收到Control/Query必须响应
   - 响应SN必须与请求SN一致

4. **目录分页**
   - 大量数据应分页传输
   - 使用SumNum表示总数