// 设备相关类型定义

/** 设备状态 */
export type DeviceStatus = '0' | '1' // 0=离线, 1=在线

/** 设备信息 */
export interface Device {
  id: number
  createdAt: string
  updatedAt: string
  deviceId: string        // 设备国标编码 (20位)
  name: string            // 设备名称
  manufacturer: string    // 厂商
  model: string           // 型号
  owner: string           // 归属
  civilCode: string       // 行政区划代码
  block: string           // 警区
  address: string         // 安装地址
  parentId: string        // 父设备 ID
  safetyWay: string       // 安全传输方式
  registerWay: string     // 注册方式
  secrecy: string         // 保密属性
  ip: string              // IP 地址
  port: number            // 端口
  status: DeviceStatus    // 状态: 0=离线, 1=在线
  longitude: number       // 经度
  latitude: number        // 纬度
  lastRegisterTime: string  // 最后注册时间
  lastKeepaliveTime: string // 最后心跳时间
  channelCount: number    // 通道数量
}

/** 设备查询参数 */
export interface DeviceQuery {
  page?: number
  pageSize?: number
  status?: DeviceStatus
}

/** 设备列表响应 */
export interface DeviceListResponse {
  total: number
  list: Device[]
}

/** 设备统计 */
export interface DeviceStats {
  total: number
  online: number
  offline: number
}

/** 通道状态 */
export type ChannelStatus = '0' | '1' // 0=离线, 1=在线

/** 通道信息 */
export interface Channel {
  id: number
  createdAt: string
  updatedAt: string
  channelId: string       // 通道国标编码 (20位)
  deviceId: string        // 所属设备 ID
  name: string            // 通道名称
  manufacturer: string    // 厂商
  model: string           // 型号
  owner: string           // 归属
  civilCode: string       // 行政区划代码
  block: string           // 警区
  address: string         // 安装地址
  parentId: string        // 父设备 ID
  safetyWay: string       // 安全传输方式
  registerWay: string     // 注册方式
  secrecy: string         // 保密属性
  ip: string              // IP 地址
  port: number            // 端口
  status: ChannelStatus   // 状态: 0=离线, 1=在线
  longitude: number       // 经度
  latitude: number        // 纬度
  ptzType: number         // PTZ 类型: 0=未知, 1=球机, 2=半球, 3=固定机, 4=遥控机
  streamType: number      // 流类型: 0=普通, 1=高清
  hasAudio: boolean       // 是否有音频
  gpsChannel: boolean     // 是否 GPS 通道
}

/** 通道查询参数 */
export interface ChannelQuery {
  page?: number
  pageSize?: number
  deviceId?: string
}

/** 通道列表响应 */
export interface ChannelListResponse {
  total: number
  list: Channel[]
}

/** 通道统计 */
export interface ChannelStats {
  total: number
  online: number
  offline: number
}
