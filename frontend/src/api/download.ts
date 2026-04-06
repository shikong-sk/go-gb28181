import { get, post } from './request'

const BASE_URL = '/download'

/** 下载 API */
export const downloadApi = {
  /** 开始录像下载 */
  startDownload: (params: DownloadRequestParams) => {
    return post<DownloadResponse>(`${BASE_URL}/start`, params)
  },

  /** 获取下载进度 */
  getProgress: (streamId: string) => {
    return get<ProgressResponse>(`${BASE_URL}/progress/${streamId}`)
  },

  /** 取消下载 */
  cancelDownload: (streamId: string) => {
    return post<void>(`${BASE_URL}/cancel/${streamId}`)
  },

  /** 获取下载会话列表 */
  getSessions: () => {
    return get<DownloadSessionResponse[]>(`${BASE_URL}/sessions`)
  },
}

/** 下载请求参数 */
export interface DownloadRequestParams {
  device_id: string
  channel_id: string
  start_time: string // 格式: yyyy-MM-dd HH:mm:ss
  end_time: string
  speed?: number // 倍速：1/2/4，默认 1
}

/** 下载响应 */
export interface DownloadResponse {
  stream_id: string
  device_id: string
  channel_id: string
  status: string
  speed: number
  progress: number
  created_at: string
}

/** 下载进度响应 */
export interface ProgressResponse {
  stream_id: string
  device_id: string
  channel_id: string
  status: string
  progress: number
  speed: number
  error?: string
  start_time: string
  end_time: string
  created_at: string
  updated_at: string
}

/** 下载会话响应 */
export interface DownloadSessionResponse {
  stream_id: string
  device_id: string
  channel_id: string
  status: string
  progress: number
  speed: number
  error?: string
  start_time: string
  end_time: string
  created_at: string
  updated_at: string
}