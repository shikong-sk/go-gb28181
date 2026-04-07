/**
 * WebSocket 事件服务
 * 用于实时接收设备状态变更、报警通知等事件
 */

/** WebSocket 事件类型 */
export type WSEventType =
  | 'device_online'      // 设备上线
  | 'device_offline'     // 设备离线
  | 'device_status'      // 设备状态更新
  | 'device_info'        // 设备信息更新
  | 'device_position'    // 设备位置更新
  | 'alarm'              // 报警通知
  | 'channel_status'     // 通道状态变更
  | 'channel_update'     // 通道信息更新
  | 'keepalive'          // 心跳更新
  | 'play_session_start' // 播放会话开始
  | 'play_session_stop'  // 播放会话停止
  | 'download_progress'  // 下载进度更新

/** WebSocket 事件数据 */
export interface WSEvent<T = unknown> {
  type: WSEventType
  data: T
  timestamp: string
}

/** 设备上线/离线事件数据 */
export interface DeviceOnlineEvent {
  device_id: string
  device_name?: string
  ip?: string
  port?: number
}

/** 设备状态事件数据 */
export interface DeviceStatusEvent {
  device_id: string
  status: string
  online: boolean
  record_status?: string
  storage_status?: string
}

/** 设备位置事件数据 */
export interface DevicePositionEvent {
  device_id: string
  longitude: number
  latitude: number
  altitude?: number
  speed?: number
  direction?: number
  timestamp: string
}

/** 报警事件数据 */
export interface AlarmEvent {
  id: number
  device_id: string
  alarm_priority: string
  alarm_method: string
  alarm_time: string
  alarm_description: string
}

/** 通道更新事件数据 */
export interface ChannelUpdateEvent {
  device_id: string
  channel_id: string
  status?: string
  name?: string
}

/** 播放会话开始事件数据 */
export interface PlaySessionStartEvent {
  stream_id: string
  device_id: string
  channel_id: string
  mode: 'live' | 'playback'
  flv_url?: string
  hls_url?: string
  rtsp_url?: string
}

/** 播放会话停止事件数据 */
export interface PlaySessionStopEvent {
  stream_id: string
  device_id: string
  channel_id: string
  reason?: string
}

/** 下载进度事件数据 */
export interface DownloadProgressEvent {
  stream_id: string
  device_id: string
  channel_id: string
  progress: number
  status: 'pending' | 'downloading' | 'completed' | 'failed' | 'cancelled'
  speed?: number
  error?: string
}

/** WebSocket 连接状态 */
export type WSConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

/** WebSocket 事件回调 */
export type WSEventCallback<T = unknown> = (event: WSEvent<T>) => void

/** WebSocket 服务类 */
class WebSocketService {
  private ws: WebSocket | null = null
  private url: string
  private status: WSConnectionStatus = 'disconnected'
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 3000
  private eventCallbacks: Map<WSEventType, Set<WSEventCallback>> = new Map()
  private statusCallbacks: Set<(status: WSConnectionStatus) => void> = new Set()
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null

  constructor() {
    // WebSocket 端点地址
    this.url = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws/events`
  }

  /** 获取连接状态 */
  getStatus(): WSConnectionStatus {
    return this.status
  }

  /** 连接 WebSocket */
  connect(): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    this.status = 'connecting'
    this.notifyStatusChange()

    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        this.status = 'connected'
        this.reconnectAttempts = 0
        this.notifyStatusChange()
        this.startHeartbeat()
        console.log('[WebSocket] 已连接')
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WSEvent = JSON.parse(event.data)
          this.handleEvent(message)
        } catch (e) {
          console.error('[WebSocket] 解析消息失败:', e)
        }
      }

      this.ws.onerror = (error) => {
        console.error('[WebSocket] 连接错误:', error)
        this.status = 'error'
        this.notifyStatusChange()
      }

      this.ws.onclose = () => {
        this.status = 'disconnected'
        this.notifyStatusChange()
        this.stopHeartbeat()
        console.log('[WebSocket] 已断开')
        this.tryReconnect()
      }
    } catch (e) {
      console.error('[WebSocket] 创建连接失败:', e)
      this.status = 'error'
      this.notifyStatusChange()
      this.tryReconnect()
    }
  }

  /** 断开连接 */
  disconnect(): void {
    this.stopHeartbeat()
    this.reconnectAttempts = this.maxReconnectAttempts // 阻止自动重连
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.status = 'disconnected'
    this.notifyStatusChange()
  }

  /** 重置重连计数（允许重新连接） */
  resetReconnect(): void {
    this.reconnectAttempts = 0
    // 如果当前有连接，先关闭
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.status = 'disconnected'
  }

  /** 尝试重连 */
  private tryReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('[WebSocket] 已达到最大重连次数，停止重连')
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * this.reconnectAttempts
    console.log(`[WebSocket] ${delay}ms 后尝试第 ${this.reconnectAttempts} 次重连`)

    setTimeout(() => {
      this.connect()
    }, delay)
  }

  /** 开始心跳 */
  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'heartbeat' }))
      }
    }, 30000)
  }

  /** 停止心跳 */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  /** 处理事件 */
  private handleEvent(event: WSEvent): void {
    const callbacks = this.eventCallbacks.get(event.type)
    if (callbacks) {
      callbacks.forEach((cb) => cb(event))
    }
  }

  /** 通知状态变更 */
  private notifyStatusChange(): void {
    this.statusCallbacks.forEach((cb) => cb(this.status))
  }

  /**订阅事件 */
  subscribe<T>(eventType: WSEventType, callback: WSEventCallback<T>): () => void {
    if (!this.eventCallbacks.has(eventType)) {
      this.eventCallbacks.set(eventType, new Set())
    }
    const callbacks = this.eventCallbacks.get(eventType)!
    callbacks.add(callback as WSEventCallback)

    // 返回取消订阅函数
    return () => {
      callbacks.delete(callback as WSEventCallback)
    }
  }

  /** 订阅连接状态变更 */
  subscribeStatus(callback: (status: WSConnectionStatus) => void): () => void {
    this.statusCallbacks.add(callback)
    // 立即通知当前状态
    callback(this.status)
    return () => {
      this.statusCallbacks.delete(callback)
    }
  }

  /** 订阅所有事件 */
  subscribeAll(callback: WSEventCallback): () => void {
    const unsubscribes: (() => void)[] = []
    const eventTypes: WSEventType[] = [
      'device_online', 'device_offline', 'device_status',
      'device_info', 'device_position', 'alarm',
      'channel_status', 'channel_update', 'keepalive',
      'play_session_start', 'play_session_stop', 'download_progress'
    ]
    eventTypes.forEach((type) => {
      unsubscribes.push(this.subscribe(type, callback))
    })
    return () => unsubscribes.forEach((unsub) => unsub())
  }
}

/** 导出单例实例 */
export const wsService = new WebSocketService()