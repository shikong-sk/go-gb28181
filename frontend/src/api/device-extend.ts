import { get, post } from './request'

/** 设备位置信息 */
export interface DevicePosition {
  device_id: string
  longitude: number
  latitude: number
  altitude?: number
  speed?: number      // 速度 km/h
  direction?: number  // 方向 0-360
  timestamp: string   // 定位时间
}

/** 设备位置历史记录 */
export interface DevicePositionRecord {
  id: number
  device_id: string
  longitude: number
  latitude: number
  altitude?: number
  speed?: number
  direction?: number
  gps_time: string
  created_at: string
}

/** 设备位置历史查询参数 */
export interface PositionHistoryQuery {
  start_time?: string
  end_time?: string
  limit?: number
}

/** 设备位置历史响应 */
export interface PositionHistoryResponse {
  device_id: string
  total: number
  list: DevicePositionRecord[]
}

/** 设备状态信息 */
export interface DeviceStatusInfo {
  device_id: string
  status: string       // 设备状态
  online: boolean      // 是否在线
  last_keepalive: string  // 最后心跳时间
  record_status?: string  // 录像状态
  storage_status?: string // 存储状态
  net_status?: string     // 网络状态
  device_time?: string    // 设备时间
}

/** 设备详细信息（查询响应） */
export interface DeviceInfoDetail {
  device_id: string
  device_name?: string
  manufacturer?: string
  model?: string
  firmware?: string    // 固件版本
  max_camera_count?: number  // 最大摄像头数
  alarm_status?: string      // 报警状态
  update_time?: string       // 更新时间
}

const BASE_URL = '/devices'

/** 设备扩展 API（位置、状态、信息查询） */
export const deviceExtendApi = {
  /** 获取设备最新位置 */
  getPosition: (deviceId: string) => {
    return get<DevicePosition>(`${BASE_URL}/${deviceId}/position`)
  },

  /** 获取设备历史轨迹 */
  getPositionHistory: (deviceId: string, params?: PositionHistoryQuery) => {
    return get<PositionHistoryResponse>(`${BASE_URL}/${deviceId}/positions`, params as Record<string, unknown>)
  },

  /** 获取设备状态（数据库缓存） */
  getStatus: (deviceId: string) => {
    return get<DeviceStatusInfo>(`${BASE_URL}/${deviceId}/status`)
  },

  /** 主动查询设备状态（发送 SIP MESSAGE） */
  queryStatus: (deviceId: string) => {
    return post<void>(`${BASE_URL}/${deviceId}/query-status`)
  },

  /** 主动查询设备信息（发送 SIP MESSAGE） */
  queryInfo: (deviceId: string) => {
    return post<void>(`${BASE_URL}/${deviceId}/query-info`)
  },

  /** 获取设备信息（数据库缓存） */
  getInfo: (deviceId: string) => {
    return get<DeviceInfoDetail>(`${BASE_URL}/${deviceId}/info`)
  },
}