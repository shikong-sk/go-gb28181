import { post } from './request'

/** PTZ 方向类型 */
export type PTZDirection = 'Stop' | 'Up' | 'Down' | 'Left' | 'Right' | 'ZoomIn' | 'ZoomOut'

/** PTZ 控制参数 */
export interface PTZControlParams {
  device_id: string
  channel_id: string
  direction: PTZDirection
  speed?: number  // 速度值 1-255，默认 128
}

/** PTZ 响应 */
export interface PTZResponse {
  success: boolean
  message?: string
}

const BASE_URL = '/ptz'

/** PTZ 云台控制 API */
export const ptzApi = {
  /** 云台控制 */
  control: (params: PTZControlParams) => {
    return post<PTZResponse>(`${BASE_URL}/control`, params)
  },

  /** 停止云台 */
  stop: (deviceId: string, channelId: string) => {
    return post<PTZResponse>(`${BASE_URL}/stop`, {
      device_id: deviceId,
      channel_id: channelId,
    })
  },
}