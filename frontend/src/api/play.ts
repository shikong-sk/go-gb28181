import { post, get } from './request'

const BASE_URL = '/api/play'

/** 播放 API */
export const playApi = {
  /** 开始播放 */
  start: (deviceId: string, channelId: string) => {
    return post<PlayResponse>(`${BASE_URL}/start`, {
      device_id: deviceId,
      channel_id: channelId,
    })
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
}

/** 播放响应 */
export interface PlayResponse {
  stream_id: string
  flv_url: string
  hls_url: string
  rtsp_url: string
  rtmp_url: string
}

/** 播放会话 */
export interface SessionResponse {
  stream_id: string
  device_id: string
  channel_id: string
  status: string
  start_time: string
  flv_url: string
  hls_url: string
}

/** 媒体信息 */
export interface MediaInfoResponse {
  has_stream: boolean
  info?: unknown
}
