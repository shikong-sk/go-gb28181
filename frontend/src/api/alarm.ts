import { get, del } from './request'

/** 报警记录 */
export interface Alarm {
  id: number
  deviceId: string
  alarmPriority: string
  alarmMethod: string
  alarmTime: string
  alarmDescription: string
  createdAt: string
}

/** 报警查询参数 */
export interface AlarmQuery {
  page?: number
  page_size?: number
  device_id?: string
  priority?: string
  start_time?: string
  end_time?: string
}

/** 报警列表响应 */
export interface AlarmListResponse {
  total: number
  list: Alarm[]
}

/** 报警配置 */
export interface AlarmConfig {
  enabled: boolean
  retention_days: number
}

const BASE_URL = '/alarms'

/** 报警 API */
export const alarmApi = {
  /** 获取报警列表 */
  getList: (params?: AlarmQuery) => {
    return get<AlarmListResponse>(BASE_URL, params as Record<string, unknown>)
  },

  /** 获取报警详情 */
  getDetail: (id: number) => {
    return get<Alarm>(`${BASE_URL}/${id}`)
  },

  /** 删除报警 */
  delete: (id: number) => {
    return del<void>(`${BASE_URL}/${id}`)
  },

  /** 清空所有报警 */
  deleteAll: () => {
    return del<void>(BASE_URL)
  },

  /** 获取报警配置 */
  getConfig: () => {
    return get<AlarmConfig>(`${BASE_URL}/config`)
  },
}