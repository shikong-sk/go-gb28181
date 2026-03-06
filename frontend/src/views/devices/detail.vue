<template>
  <div class="tw-p-6">
    <!-- 页面标题 -->
    <div class="tw-mb-6">
      <el-page-header @back="goBack">
        <template #content>
          <span class="tw-text-xl tw-font-bold">设备详情</span>
        </template>
      </el-page-header>
    </div>

    <div v-loading="deviceStore.loading">
      <!-- 基本信息 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
        <h3 class="tw-text-lg tw-font-bold tw-mb-4 tw-border-b tw-pb-2">基本信息</h3>
        <div class="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4">
          <div>
            <label class="tw-text-sm tw-text-gray-500">设备ID</label>
            <p class="tw-font-mono">{{ device?.deviceId }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">设备名称</label>
            <p>{{ device?.name || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">状态</label>
            <p>
              <el-tag :type="device?.status === '1' ? 'success' : 'danger'">
                {{ device?.status === '1' ? '在线' : '离线' }}
              </el-tag>
            </p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">厂商</label>
            <p>{{ device?.manufacturer || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">型号</label>
            <p>{{ device?.model || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">归属</label>
            <p>{{ device?.owner || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">IP地址</label>
            <p>{{ device?.ip }}:{{ device?.port }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">行政区划</label>
            <p>{{ device?.civilCode || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">安装地址</label>
            <p>{{ device?.address || '-' }}</p>
          </div>
        </div>
      </div>

      <!-- 状态信息 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
        <h3 class="tw-text-lg tw-font-bold tw-mb-4 tw-border-b tw-pb-2">状态信息</h3>
        <div class="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4">
          <div>
            <label class="tw-text-sm tw-text-gray-500">最后注册时间</label>
            <p>{{ formatTime(device?.lastRegisterTime) }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">最后心跳时间</label>
            <p>{{ formatTime(device?.lastKeepaliveTime) }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">通道数量</label>
            <p>{{ device?.channelCount || 0 }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">经度</label>
            <p>{{ device?.longitude || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">纬度</label>
            <p>{{ device?.latitude || '-' }}</p>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
        <h3 class="tw-text-lg tw-font-bold tw-mb-4 tw-border-b tw-pb-2">操作</h3>
        <div class="tw-flex tw-gap-4">
          <el-button type="primary" @click="viewChannels">
            查看通道列表
          </el-button>
          <el-popconfirm
            title="确定删除该设备？此操作将同时删除该设备的所有通道数据。"
            confirm-button-text="确定"
            cancel-button-text="取消"
            @confirm="deleteDevice"
          >
            <template #reference>
              <el-button type="danger">删除设备</el-button>
            </template>
          </el-popconfirm>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '@/stores/device'

const route = useRoute()
const router = useRouter()
const deviceStore = useDeviceStore()

const deviceId = computed(() => route.params.deviceId as string)
const device = computed(() => deviceStore.currentDevice)

/** 格式化时间 */
function formatTime(time: string | undefined) {
  if (!time || time === '0001-01-01T00:00:00Z') return '-'
  return new Date(time).toLocaleString('zh-CN')
}

/** 返回上一页 */
function goBack() {
  router.push('/devices')
}

/** 查看通道列表 */
function viewChannels() {
  router.push({ path: '/channels', query: { deviceId: deviceId.value } })
}

/** 删除设备 */
async function deleteDevice() {
  const success = await deviceStore.deleteDevice(deviceId.value)
  if (success) {
    ElMessage.success('删除成功')
    goBack()
  }
}

onMounted(() => {
  if (deviceId.value) {
    deviceStore.fetchDevice(deviceId.value)
  }
})
</script>