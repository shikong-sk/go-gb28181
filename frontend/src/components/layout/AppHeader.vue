<template>
  <div class="tw-h-full tw-flex tw-items-center tw-justify-between">
    <h2 class="tw-text-lg tw-font-medium">{{ $route.meta.title || 'GB28181平台' }}</h2>
    <div class="tw-flex tw-items-center tw-gap-4">
      <!-- WebSocket 连接状态指示器 -->
      <el-tag
        :type="deviceStore.wsConnected ? 'success' : 'warning'"
        size="small"
        class="tw-cursor-pointer"
        @click="toggleWebSocket"
      >
        <el-icon class="tw-mr-1"><Connection /></el-icon>
        {{ deviceStore.wsConnected ? 'WebSocket 已连接' : 'WebSocket 未连接' }}
      </el-tag>
      <el-button type="primary" size="small" @click="refreshDevices">
        <el-icon class="tw-mr-1"><Refresh /></el-icon>
        刷新设备
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Refresh, Connection } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '@/stores/device'

const deviceStore = useDeviceStore()

const refreshDevices = () => {
  deviceStore.fetchDevices()
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
</script>