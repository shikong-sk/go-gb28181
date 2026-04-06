import { get, del, post } from './request'

/** 目录订阅信息 */
export interface CatalogSubscription {
  device_id: string
  sn: string
  event_id: string
  expires: number
  subscribed_at: string
}

/** 目录订阅列表响应 */
export interface CatalogSubscriptionListResponse {
  total: number
  list: CatalogSubscription[]
}

/** 报警订阅信息 */
export interface AlarmSubscription {
  device_id: string
  sn: string
  subscribe_id: string
  expires: number
  subscribed_at: string
}

/** 报警订阅列表响应 */
export interface AlarmSubscriptionListResponse {
  total: number
  subscriptions: AlarmSubscription[]
}

/** 订阅操作响应 */
export interface SubscriptionResponse {
  message: string
  device_id?: string
  expires?: number
  sn?: string
  count?: number
}

/** 订阅查询参数 */
export interface SubscriptionQuery {
  expires?: number // 过期时间（秒），默认 3600
  sn?: string      // 订阅 SN，用于取消指定订阅
}

/** 目录订阅 API */
export const catalogSubscriptionApi = {
  /** 获取所有目录订阅列表 */
  getList: () => {
    return get<CatalogSubscriptionListResponse>('/subscriptions')
  },

  /** 订阅设备目录 */
  subscribe: (deviceId: string, expires?: number) => {
    const params: SubscriptionQuery = {}
    if (expires) {
      params.expires = expires
    }
    return post<SubscriptionResponse>(`/devices/${deviceId}/subscribe/catalog`, null, { params })
  },

  /** 取消目录订阅 */
  unsubscribe: (deviceId: string, sn?: string) => {
    const params: SubscriptionQuery = {}
    if (sn) {
      params.sn = sn
    }
    return del<SubscriptionResponse>(`/devices/${deviceId}/subscribe/catalog`, params)
  },
}

/** 报警订阅 API */
export const alarmSubscriptionApi = {
  /** 获取所有报警订阅列表 */
  getList: () => {
    return get<AlarmSubscriptionListResponse>('/alarms/subscriptions')
  },

  /** 订阅设备报警 */
  subscribe: (deviceId: string, expires?: number) => {
    const params: SubscriptionQuery = {}
    if (expires) {
      params.expires = expires
    }
    return post<SubscriptionResponse>(`/devices/${deviceId}/subscribe/alarm`, null, { params })
  },

  /** 取消报警订阅 */
  unsubscribe: (deviceId: string, sn: string) => {
    return del<SubscriptionResponse>(`/devices/${deviceId}/subscribe/alarm`, { sn })
  },

  /** 全局报警订阅（订阅所有在线设备） */
  subscribeGlobal: (expires?: number) => {
    const params: SubscriptionQuery = {}
    if (expires) {
      params.expires = expires
    }
    return post<SubscriptionResponse>('/alarms/subscribe', null, { params })
  },
}

/** 统一的订阅管理 API */
export const subscriptionApi = {
  catalog: catalogSubscriptionApi,
  alarm: alarmSubscriptionApi,

  /** 获取所有订阅状态（合并目录和报警订阅） */
  getAllSubscriptions: async () => {
    const [catalogRes, alarmRes] = await Promise.all([
      catalogSubscriptionApi.getList(),
      alarmSubscriptionApi.getList(),
    ])
    return {
      catalog: catalogRes.data?.list || [],
      alarm: alarmRes.data?.subscriptions || [],
    }
  },
}