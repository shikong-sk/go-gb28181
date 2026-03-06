import { get, del, post } from './request'
import type {
  Device,
  DeviceQuery,
  DeviceListResponse,
  DeviceStats,
  Channel,
  ChannelQuery,
  ChannelListResponse,
  ChannelStats,
} from '../types/device'

const BASE_URL = '/devices'
const CHANNEL_URL = '/channels'

/** 设备 API */
export const deviceApi = {
  /** 获取设备列表 */
  getList: (params?: DeviceQuery) => {
    return get<DeviceListResponse>(BASE_URL, params as Record<string, unknown>)
  },

  /** 获取设备详情 */
  getDetail: (deviceId: string) => {
    return get<Device>(`${BASE_URL}/${deviceId}`)
  },

  /** 获取设备统计 */
  getStats: () => {
    return get<DeviceStats>(`${BASE_URL}/stats`)
  },

  /** 删除设备 */
  delete: (deviceId: string) => {
    return del<void>(`${BASE_URL}/${deviceId}`)
  },

  /** 同步设备目录 */
  syncCatalog: (deviceId: string) => {
    return post<void>(`${BASE_URL}/${deviceId}/sync`)
  },
}

/** 通道 API */
export const channelApi = {
  /** 获取通道列表 */
  getList: (params?: ChannelQuery) => {
    return get<ChannelListResponse>(CHANNEL_URL, params as Record<string, unknown>)
  },

  /** 获取通道详情 */
  getDetail: (channelId: string, deviceId: string) => {
    return get<Channel>(`${CHANNEL_URL}/${channelId}`, { deviceId })
  },

  /** 获取通道统计 */
  getStats: (deviceId?: string) => {
    return get<ChannelStats>(`${CHANNEL_URL}/stats`, deviceId ? { deviceId } : undefined)
  },

  /** 删除通道 */
  delete: (channelId: string, deviceId: string) => {
    return del<void>(`${CHANNEL_URL}/${channelId}`, { deviceId })
  },
}
