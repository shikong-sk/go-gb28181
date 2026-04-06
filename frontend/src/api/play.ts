import { post, get } from './request'

const BASE_URL = '/play'
const RECORD_BASE_URL = '/record'

/** 播放模式 */
export type PlayMode = 'live' | 'playback'

/** 播放 API */
export const playApi = {
  /** 开始实时播放 */
  start: (deviceId: string, channelId: string) => {
    return post<PlayResponse>(`${BASE_URL}/start`, {
      device_id: deviceId,
      channel_id: channelId,
    })
  },

  /** 开始录像回放 */
  startPlayback: (params: PlayBackParams) => {
    return post<PlayResponse>(`${BASE_URL}/playback`, params)
  },

  /** 停止播放 */
  stop: (streamId: string) => {
    return post<void>(`${BASE_URL}/stop`, {
      stream_id: streamId,
    })
  },

  /** 获取播放会话列表 */
  getSessions: () => {
    return get<SessionResponse[]>(`${BASE_URL}/sessions`)
  },

  /** 获取媒体信息 */
  getMediaInfo: (streamId: string) => {
    return get<MediaInfoResponse>(`${BASE_URL}/media/${streamId}`)
  },

  /** 查询历史录像 */
  queryRecords: (params: RecordQueryParams) => {
    return get<RecordListResponse>(`${RECORD_BASE_URL}/list`, params as unknown as Record<string, unknown>)
  },

  /** 触发录像拉取（强制刷新缓存） */
  fetchRecords: (params: FetchRecordsParams) => {
    return post<FetchRecordsResponse>(`${RECORD_BASE_URL}/fetch`, params)
  },

  /** 获取录像拉取状态 */
  getFetchStatus: (deviceId: string, channelId: string, date?: string) => {
    const params: Record<string, unknown> = {
      device_id: deviceId,
      channel_id: channelId,
    }
    if (date) {
      params.date = date
    }
    return get<FetchRecordsResponse>(`${RECORD_BASE_URL}/fetch/status`, params)
  },
}

/** 回放请求参数 */
export interface PlayBackParams {
  device_id: string
  channel_id: string
  start_time: string
  end_time: string
}

/** 播放响应 */
export interface PlayResponse {
  stream_id: string
  urls: string[]
  rtp_port: number
  flv_url: string
  hls_url: string
  rtsp_url: string
  rtmp_url: string
  mode: PlayMode
}

/** 播放会话 */
export interface SessionResponse {
  stream_id: string
  device_id: string
  channel_id: string
  rtp_port: number
  mode: PlayMode
  status: string
  start_time: string
  playback_start?: string
  playback_end?: string
}

/** 媒体信息 */
export interface MediaInfoResponse {
  has_stream: boolean
  info?: unknown
}

/** 录像查询参数 */
export interface RecordQueryParams {
  device_id: string
  channel_id: string
  date?: string
  timeout?: number
  force?: boolean  // 强制刷新缓存
}

/** 录像记录项 */
export interface RecordItem {
  device_id: string
  name: string
  address: string
  start_time: string
  end_time: string
  secrecy: number
  type: string
  file_size: number
}

/** 录像查询响应 */
export interface RecordListResponse {
  device_id: string
  channel_id: string
  total: number
  items: RecordItem[]
  source: string      // 数据来源: cache, db, device
  cached_at: string   // 缓存时间（仅缓存数据有）
  expires_at: string  // 过期时间（仅缓存数据有）
  expires_in: number  // 距离过期剩余秒数（仅缓存数据有）
  item_count: number  // 录像项数量
}

/** 录像拉取请求 */
export interface FetchRecordsParams {
  device_id: string
  channel_id: string
  date?: string
  timeout?: number
}

/** 录像拉取响应 */
export interface FetchRecordsResponse {
  device_id: string
  channel_id: string
  date: string
  status: string     // 拉取状态: pending, fetching, completed, failed, expired
  message: string    // 状态消息
  item_count: number // 已拉取的录像数量
}
