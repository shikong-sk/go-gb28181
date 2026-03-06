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
      <div class="tw-flex tw-gap-4">
        <el-button type="primary" @click="$router.push('/devices')">
          <el-icon class="tw-mr-1"><Monitor /></el-icon>
          设备管理
        </el-button>
        <el-button type="success" @click="$router.push('/channels')">
          <el-icon class="tw-mr-1"><VideoCamera /></el-icon>
          通道管理
        </el-button>
        <el-button @click="handleRefresh">
          <el-icon class="tw-mr-1"><Refresh /></el-icon>
          刷新数据
        </el-button>
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
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Monitor, CircleCheck, CircleClose, VideoCamera, Refresh } from '@element-plus/icons-vue'
import { useDeviceStore, useChannelStore } from '@/stores/device'

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

onMounted(() => {
  deviceStore.fetchDevices()
  deviceStore.fetchStats()
  channelStore.fetchStats()
})
</script>