<template>
  <div class="tw-h-full tw-flex tw-items-center tw-justify-between">
    <h2 class="tw-text-lg tw-font-medium">{{ $route.meta.title || 'GB28181平台' }}</h2>
    <div class="tw-flex tw-items-center tw-gap-4">
      <el-button type="primary" size="small" @click="refreshDevices">
        <el-icon class="tw-mr-1"><Refresh /></el-icon>
        刷新设备
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Refresh } from '@element-plus/icons-vue'
import { useDeviceStore } from '@/stores/device'

const deviceStore = useDeviceStore()

const refreshDevices = () => {
  deviceStore.fetchDevices()
}
</script>