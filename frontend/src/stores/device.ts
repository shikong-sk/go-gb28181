import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElNotification } from 'element-plus'
import { deviceApi, channelApi } from '../api/device'
import type { Device, DeviceQuery, DeviceStats, Channel, ChannelQuery, ChannelStats } from '../types/device'
import { wsService, type WSEvent, type DeviceOnlineEvent, type DeviceStatusEvent, type AlarmEvent, type PlaySessionStartEvent, type PlaySessionStopEvent } from '../api/websocket'

/** 设备状态管理 */
export const useDeviceStore = defineStore('device', () => {
  // 状态
  const devices = ref<Device[]>([])
  const currentDevice = ref<Device | null>(null)
  const stats = ref<DeviceStats>({ total: 0, online: 0, offline: 0 })
  const loading = ref(false)
  const total = ref(0)

  // WebSocket 连接状态
  const wsConnected = ref(false)

  // 实时事件列表（用于 dashboard 展示）
  const recentEvents = ref<WSEvent[]>([])

  // 分页参数
  const pagination = ref({
    page: 1,
    pageSize: 20,
  })

  // 筛选参数
  const filter = ref<DeviceQuery>({})

  // 计算属性
  const onlineDevices = computed(() => devices.value.filter((d) => d.status === '1'))
  const offlineDevices = computed(() => devices.value.filter((d) => d.status === '0'))

  /** 获取设备列表 */
  async function fetchDevices(params?: DeviceQuery) {
    loading.value = true
    try {
      const query: DeviceQuery = {
        page: pagination.value.page,
        pageSize: pagination.value.pageSize,
        ...filter.value,
        ...params,
      }
      const res = await deviceApi.getList(query)
      if (res.data) {
        devices.value = res.data.list || []
        total.value = res.data.total || 0
      }
    } catch (error) {
      console.error('获取设备列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  /** 获取设备详情 */
  async function fetchDevice(deviceId: string) {
    loading.value = true
    try {
      const res = await deviceApi.getDetail(deviceId)
      if (res.data) {
        currentDevice.value = res.data
      }
    } catch (error) {
      console.error('获取设备详情失败:', error)
    } finally {
      loading.value = false
    }
  }

  /** 获取设备统计 */
  async function fetchStats() {
    try {
      const res = await deviceApi.getStats()
      if (res.data) {
        stats.value = res.data
      }
    } catch (error) {
      console.error('获取设备统计失败:', error)
    }
  }

  /** 同步设备目录 */
  async function syncCatalog(deviceId: string) {
    try {
      await deviceApi.syncCatalog(deviceId)
      return true
    } catch (error) {
      console.error('同步设备目录失败:', error)
      return false
    }
  }

  /** 删除设备 */
  async function deleteDevice(deviceId: string) {
    try {
      await deviceApi.delete(deviceId)
      // 刷新列表
      await fetchDevices()
      await fetchStats()
      return true
    } catch (error) {
      console.error('删除设备失败:', error)
      return false
    }
  }

  /** 更新分页参数 */
  function updatePagination(page: number, pageSize: number) {
    pagination.value.page = page
    pagination.value.pageSize = pageSize
    fetchDevices()
  }

  /** 更新筛选参数 */
  function updateFilter(newFilter: DeviceQuery) {
    filter.value = { ...newFilter }
    pagination.value.page = 1 // 重置到第一页
    fetchDevices()
  }

  /** 搜索设备（按关键词） */
  function searchDevices(keyword: string) {
    filter.value.keyword = keyword
    pagination.value.page = 1
    fetchDevices()
  }

  /** 处理设备上线事件 */
  function handleDeviceOnline(event: WSEvent<DeviceOnlineEvent>) {
    const data = event.data
    // 更新设备列表中的状态
    const device = devices.value.find(d => d.deviceId === data.device_id)
    if (device) {
      device.status = '1'
    }
    // 更新统计
    stats.value.online++
    stats.value.offline--
    // 添加到事件列表
    addEvent(event)
    // 显示通知
    ElNotification({
      title: '设备上线',
      message: `设备 ${data.device_name || data.device_id} 已上线`,
      type: 'success',
      duration: 3000,
    })
  }

  /** 处理设备离线事件 */
  function handleDeviceOffline(event: WSEvent<DeviceOnlineEvent>) {
    const data = event.data
    // 更新设备列表中的状态
    const device = devices.value.find(d => d.deviceId === data.device_id)
    if (device) {
      device.status = '0'
    }
    // 更新统计
    stats.value.online--
    stats.value.offline++
    // 添加到事件列表
    addEvent(event)
    // 显示通知
    ElNotification({
      title: '设备离线',
      message: `设备 ${data.device_name || data.device_id} 已离线`,
      type: 'warning',
      duration: 3000,
    })
  }

  /** 处理设备状态更新事件 */
  function handleDeviceStatus(event: WSEvent<DeviceStatusEvent>) {
    const data = event.data
    // 更新当前设备状态
    if (currentDevice.value && currentDevice.value.deviceId === data.device_id) {
      // 可以扩展 Device 类型来包含更多状态信息
    }
    // 添加到事件列表
    addEvent(event)
  }

  /** 处理报警事件 */
  function handleAlarm(event: WSEvent<AlarmEvent>) {
    // 添加到事件列表
    addEvent(event)
    // 显示通知
    const priorityText = event.data.alarm_priority === '1' ? '高' : event.data.alarm_priority === '2' ? '中' : '低'
    ElNotification({
      title: '报警通知',
      message: `设备 ${event.data.device_id} ${priorityText}优先级报警: ${event.data.alarm_description}`,
      type: 'warning',
      duration: 5000,
    })
  }

  /** 处理播放会话开始事件 */
  function handlePlaySessionStart(event: WSEvent<PlaySessionStartEvent>) {
    // 添加到事件列表
    addEvent(event)
    // 显示通知
    const modeText = event.data.mode === 'live' ? '实时预览' : '录像回放'
    ElNotification({
      title: '播放会话开始',
      message: `${modeText} ${event.data.device_id}/${event.data.channel_id}`,
      type: 'info',
      duration: 2000,
    })
  }

  /** 处理播放会话停止事件 */
  function handlePlaySessionStop(event: WSEvent<PlaySessionStopEvent>) {
    // 添加到事件列表
    addEvent(event)
    // 显示通知
    ElNotification({
      title: '播放会话停止',
      message: `会话 ${event.data.stream_id} 已停止${event.data.reason ? `: ${event.data.reason}` : ''}`,
      type: 'info',
      duration: 2000,
    })
  }

  /** 添加事件到列表（最多保留 50 条） */
  function addEvent(event: WSEvent) {
    recentEvents.value.unshift(event)
    if (recentEvents.value.length > 50) {
      recentEvents.value.pop()
    }
  }

  // 订阅取消函数集合（用于防止重复订阅）
  let unsubscribeFunctions: (() => void)[] = []

  /** 连接 WebSocket（允许重新连接） */
  function connectWebSocket() {
    // 如果已连接则跳过
    if (wsConnected.value) {
      return
    }

    // 强制重置重连计数，允许重新连接
    wsService.resetReconnect()

    // 先取消之前的订阅，防止重复订阅
    unsubscribeFunctions.forEach(unsub => unsub())
    unsubscribeFunctions = []

    wsService.connect()

    // 订阅连接状态
    unsubscribeFunctions.push(
      wsService.subscribeStatus((status) => {
        wsConnected.value = status === 'connected'
      })
    )

    // 订阅设备上线事件
    unsubscribeFunctions.push(
      wsService.subscribe('device_online', handleDeviceOnline)
    )

    // 订阅设备离线事件
    unsubscribeFunctions.push(
      wsService.subscribe('device_offline', handleDeviceOffline)
    )

    // 订阅设备状态事件
    unsubscribeFunctions.push(
      wsService.subscribe('device_status', handleDeviceStatus)
    )

    // 订阅报警事件
    unsubscribeFunctions.push(
      wsService.subscribe('alarm', handleAlarm)
    )

    // 订阅播放会话开始事件
    unsubscribeFunctions.push(
      wsService.subscribe('play_session_start', handlePlaySessionStart)
    )

    // 订阅播放会话停止事件
    unsubscribeFunctions.push(
      wsService.subscribe('play_session_stop', handlePlaySessionStop)
    )
  }

  /** 断开 WebSocket */
  function disconnectWebSocket() {
    // 取消所有订阅
    unsubscribeFunctions.forEach(unsub => unsub())
    unsubscribeFunctions = []
    wsService.disconnect()
    wsConnected.value = false
  }

  /** 清空事件列表 */
  function clearEvents() {
    recentEvents.value = []
  }

  return {
    // 状态
    devices,
    currentDevice,
    stats,
    loading,
    total,
    pagination,
    filter,
    wsConnected,
    recentEvents,
    // 计算属性
    onlineDevices,
    offlineDevices,
    // 方法
    fetchDevices,
    fetchDevice,
    fetchStats,
    syncCatalog,
    deleteDevice,
    updatePagination,
    updateFilter,
    searchDevices,
    connectWebSocket,
    disconnectWebSocket,
    clearEvents,
  }
})

/** 通道状态管理 */
export const useChannelStore = defineStore('channel', () => {
  // 状态
  const channels = ref<Channel[]>([])
  const stats = ref<ChannelStats>({ total: 0, online: 0, offline: 0 })
  const loading = ref(false)
  const total = ref(0)

  // 分页参数
  const pagination = ref({
    page: 1,
    pageSize: 20,
  })

  // 筛选参数
  const filter = ref<ChannelQuery>({})

  /** 获取通道列表 */
  async function fetchChannels() {
    loading.value = true
    try {
      const params: ChannelQuery = {
        page: pagination.value.page,
        pageSize: pagination.value.pageSize,
        ...filter.value,
      }
      const res = await channelApi.getList(params)
      if (res.data) {
        channels.value = res.data.list || []
        total.value = res.data.total || 0
      }
    } catch (error) {
      console.error('获取通道列表失败:', error)
    } finally {
      loading.value = false
    }
  }

  /** 获取通道统计 */
  async function fetchStats(deviceId?: string) {
    try {
      const res = await channelApi.getStats(deviceId)
      if (res.data) {
        stats.value = res.data
      }
    } catch (error) {
      console.error('获取通道统计失败:', error)
    }
  }

  /** 删除通道 */
  async function deleteChannel(channelId: string, deviceId: string) {
    try {
      await channelApi.delete(channelId, deviceId)
      await fetchChannels()
      await fetchStats(deviceId)
      return true
    } catch (error) {
      console.error('删除通道失败:', error)
      return false
    }
  }

  /** 更新分页参数 */
  function updatePagination(page: number, pageSize: number) {
    pagination.value.page = page
    pagination.value.pageSize = pageSize
    fetchChannels()
  }

  /** 更新筛选参数 */
  function updateFilter(newFilter: ChannelQuery) {
    filter.value = { ...newFilter }
    pagination.value.page = 1
    fetchChannels()
  }

  /** 搜索通道（按关键词） */
  function searchChannels(keyword: string) {
    filter.value.keyword = keyword
    pagination.value.page = 1
    fetchChannels()
  }

  return {
    // 状态
    channels,
    stats,
    loading,
    total,
    pagination,
    filter,
    // 方法
    fetchChannels,
    fetchStats,
    deleteChannel,
    updatePagination,
    updateFilter,
    searchChannels,
  }
})