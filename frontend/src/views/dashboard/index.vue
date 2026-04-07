<template>
  <div class="tw-p-6">
    <!-- 页面标题 -->
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">控制台</h1>
      <p class="tw-text-gray-500 tw-mt-1">GB28181 设备管理平台</p>
    </div>

    <!-- 统计卡片 -->
    <div class="tw-grid tw-grid-cols-1 md:tw-grid-cols-4 tw-gap-4 tw-mb-6">
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-blue-100 tw-text-blue-600">
            <el-icon size="24"><Monitor /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">设备总数</p>
            <p class="tw-text-2xl tw-font-bold tw-text-gray-800">{{ deviceStore.stats.total }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-green-100 tw-text-green-600">
            <el-icon size="24"><CircleCheck /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">在线设备</p>
            <p class="tw-text-2xl tw-font-bold tw-text-green-600">{{ deviceStore.stats.online }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-red-100 tw-text-red-600">
            <el-icon size="24"><CircleClose /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">离线设备</p>
            <p class="tw-text-2xl tw-font-bold tw-text-red-600">{{ deviceStore.stats.offline }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-purple-100 tw-text-purple-600">
            <el-icon size="24"><VideoCamera /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">通道总数</p>
            <p class="tw-text-2xl tw-font-bold tw-text-gray-800">{{ channelStore.stats.total }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 快捷操作 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <h3 class="tw-text-lg tw-font-bold tw-mb-4">快捷操作</h3>
      <div class="tw-flex tw-gap-4 tw-flex-wrap">
        <el-button type="primary" @click="$router.push('/devices')">
          <el-icon class="tw-mr-1"><Monitor /></el-icon>
          设备管理
        </el-button>
        <el-button type="success" @click="$router.push('/channels')">
          <el-icon class="tw-mr-1"><VideoCamera /></el-icon>
          通道管理
        </el-button>
        <el-button @click="$router.push('/play')">
          <el-icon class="tw-mr-1"><VideoPlay /></el-icon>
          视频播放
        </el-button>
        <el-button @click="$router.push('/alarms')">
          <el-icon class="tw-mr-1"><Bell /></el-icon>
          报警管理
        </el-button>
        <el-button @click="handleRefresh">
          <el-icon class="tw-mr-1"><Refresh /></el-icon>
          刷新数据
        </el-button>
        <el-button :type="deviceStore.wsConnected ? 'success' : 'warning'" @click="toggleWebSocket">
          <el-icon class="tw-mr-1"><Connection /></el-icon>
          {{ deviceStore.wsConnected ? 'WebSocket 已连接' : 'WebSocket 未连接' }}
        </el-button>
      </div>
    </div>

    <!-- 实时事件与最近设备并排布局 -->
    <div class="tw-grid tw-grid-cols-1 lg:tw-grid-cols-2 tw-gap-6">
      <!-- 实时事件 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
        <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
          <h3 class="tw-text-lg tw-font-bold">实时事件</h3>
          <el-button size="small" @click="deviceStore.clearEvents()" :disabled="deviceStore.recentEvents.length === 0">
            清空
          </el-button>
        </div>
        <div v-if="deviceStore.recentEvents.length === 0" class="tw-text-gray-400 tw-text-center tw-py-8">
          <el-icon size="48" class="tw-mb-2"><Bell /></el-icon>
          <p>暂无实时事件</p>
          <p class="tw-text-sm tw-mt-1">WebSocket 连接后将实时推送设备状态变更、报警等事件</p>
        </div>
        <div v-else class="tw-space-y-2 tw-max-h-96 tw-overflow-y-auto">
          <div
            v-for="event in deviceStore.recentEvents"
            :key="event.timestamp"
            class="tw-flex tw-items-center tw-p-3 tw-rounded-lg tw-bg-gray-50"
          >
            <div class="tw-mr-3">
              <el-tag :type="getEventTagType(event.type)" size="small">
                {{ getEventLabel(event.type) }}
              </el-tag>
            </div>
            <div class="tw-flex-grow">
              <p class="tw-text-sm tw-font-medium">{{ getEventDescription(event) }}</p>
              <p class="tw-text-xs tw-text-gray-400">{{ formatTime(event.timestamp) }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 最近设备 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
        <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
          <h3 class="tw-text-lg tw-font-bold">最近设备</h3>
          <el-button type="primary" link @click="$router.push('/devices')">
            查看全部
          </el-button>
        </div>
        <el-table
          v-loading="deviceStore.loading"
          :data="recentDevices"
          stripe
          style="width: 100%"
        >
          <el-table-column prop="deviceId" label="设备ID" min-width="180" />
          <el-table-column prop="name" label="设备名称" min-width="150">
            <template #default="{ row }">
              {{ row.name || '-' }}
            </template>
          </el-table-column>
          <el-table-column prop="ip" label="IP地址" min-width="140">
            <template #default="{ row }">
              {{ row.ip }}:{{ row.port }}
            </template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === '1' ? 'success' : 'danger'" size="small">
                {{ row.status === '1' ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="channelCount" label="通道数" width="80" align="center" />
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Monitor, CircleCheck, CircleClose, VideoCamera, Refresh, Bell, Connection, VideoPlay } from '@element-plus/icons-vue'
import { useDeviceStore, useChannelStore } from '@/stores/device'
import type { WSEvent } from '@/api/websocket'

const deviceStore = useDeviceStore()
const channelStore = useChannelStore()

// 最近5个设备
const recentDevices = computed(() => deviceStore.devices.slice(0, 5))

/** 刷新数据 */
async function handleRefresh() {
  await Promise.all([
    deviceStore.fetchDevices(),
    deviceStore.fetchStats(),
    channelStore.fetchStats(),
  ])
  ElMessage.success('刷新成功')
}

/** 切换 WebSocket 连接 */
function toggleWebSocket() {
  if (deviceStore.wsConnected) {
    deviceStore.disconnectWebSocket()
    ElMessage.info('WebSocket 已断开')
  } else {
    deviceStore.connectWebSocket()
    ElMessage.info('正在连接 WebSocket...')
  }
}

/** 获取事件标签类型 */
function getEventTagType(type: string): 'success' | 'danger' | 'warning' | 'info' {
  switch (type) {
    case 'device_online':
      return 'success'
    case 'device_offline':
      return 'danger'
    case 'alarm':
      return 'warning'
    default:
      return 'info'
  }
}

/** 获取事件标签文本 */
function getEventLabel(type: string): string {
  switch (type) {
    case 'device_online':
      return '设备上线'
    case 'device_offline':
      return '设备离线'
    case 'device_status':
      return '状态更新'
    case 'device_info':
      return '信息更新'
    case 'device_position':
      return '位置更新'
    case 'alarm':
      return '报警'
    case 'channel_status':
      return '通道状态'
    case 'keepalive':
      return '心跳'
    default:
      return '未知'
  }
}

/** 获取事件描述 */
function getEventDescription(event: WSEvent): string {
  const data = event.data as Record<string, unknown>
  switch (event.type) {
    case 'device_online':
    case 'device_offline':
      return `设备 ${data.device_id || '未知'} ${event.type === 'device_online' ? '已上线' : '已离线'}`
    case 'alarm':
      return `设备 ${data.device_id || '未知'} 发生报警: ${data.alarm_description || '无描述'}`
    case 'device_position':
      return `设备 ${data.device_id || '未知'} 位置更新`
    default:
      return `设备 ${data.device_id || '未知'} 状态变更`
  }
}

/** 格式化时间 */
function formatTime(timestamp: string): string {
  return new Date(timestamp).toLocaleString('zh-CN')
}

onMounted(() => {
  deviceStore.fetchDevices()
  deviceStore.fetchStats()
  channelStore.fetchStats()
  // WebSocket 连接已移至 App.vue 全局管理
})

onUnmounted(() => {
  // WebSocket 断开已移至 App.vue 全局管理
})
</script>