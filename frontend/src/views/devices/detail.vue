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

      <!-- 设备位置信息 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
        <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
          <h3 class="tw-text-lg tw-font-bold tw-border-b tw-pb-2 tw-flex-grow">设备位置</h3>
          <el-button size="small" @click="refreshPosition" :loading="loadingPosition">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            刷新位置
          </el-button>
        </div>
        <div v-if="position" class="tw-grid tw-grid-cols-1 md:tw-grid-cols-4 tw-gap-4">
          <div>
            <label class="tw-text-sm tw-text-gray-500">经度</label>
            <p>{{ position.longitude.toFixed(6) }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">纬度</label>
            <p>{{ position.latitude.toFixed(6) }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">速度</label>
            <p>{{ position.speed ? `${position.speed.toFixed(2)} km/h` : '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">定位时间</label>
            <p>{{ formatTime(position.timestamp) }}</p>
          </div>
          <div class="md:tw-col-span-4">
            <el-button
              type="primary"
              link
              @click="openMap"
              :disabled="!position?.longitude || !position?.latitude"
            >
              <el-icon class="tw-mr-1"><Location /></el-icon>
              在地图中查看
            </el-button>
          </div>
        </div>
        <div v-else class="tw-text-gray-400 tw-text-center tw-py-4">
          暂无位置数据
        </div>
      </div>

      <!-- 设备状态查询 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
        <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
          <h3 class="tw-text-lg tw-font-bold tw-border-b tw-pb-2 tw-flex-grow">设备状态</h3>
          <el-button size="small" @click="queryDeviceStatus" :loading="queryingStatus">
            <el-icon class="tw-mr-1"><Search /></el-icon>
            查询实时状态
          </el-button>
        </div>
        <div v-if="deviceStatus" class="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4">
          <div>
            <label class="tw-text-sm tw-text-gray-500">在线状态</label>
            <p>
              <el-tag :type="deviceStatus.online ? 'success' : 'danger'" size="small">
                {{ deviceStatus.online ? '在线' : '离线' }}
              </el-tag>
            </p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">录像状态</label>
            <p>{{ deviceStatus.record_status || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">存储状态</label>
            <p>{{ deviceStatus.storage_status || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">网络状态</label>
            <p>{{ deviceStatus.net_status || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">最后心跳</label>
            <p>{{ formatTime(deviceStatus.last_keepalive) }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">设备时间</label>
            <p>{{ formatTime(deviceStatus.device_time) }}</p>
          </div>
        </div>
        <div v-else class="tw-text-gray-400 tw-text-center tw-py-4">
          点击"查询实时状态"获取设备最新状态
        </div>
      </div>

      <!-- 设备信息查询 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
        <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
          <h3 class="tw-text-lg tw-font-bold tw-border-b tw-pb-2 tw-flex-grow">设备详细信息</h3>
          <el-button size="small" @click="queryDeviceInfo" :loading="queryingInfo">
            <el-icon class="tw-mr-1"><Search /></el-icon>
            查询设备信息
          </el-button>
        </div>
        <div v-if="deviceInfo" class="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4">
          <div>
            <label class="tw-text-sm tw-text-gray-500">设备名称</label>
            <p>{{ deviceInfo.device_name || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">厂商</label>
            <p>{{ deviceInfo.manufacturer || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">型号</label>
            <p>{{ deviceInfo.model || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">固件版本</label>
            <p>{{ deviceInfo.firmware || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">最大摄像头数</label>
            <p>{{ deviceInfo.max_camera_count || '-' }}</p>
          </div>
          <div>
            <label class="tw-text-sm tw-text-gray-500">报警状态</label>
            <p>{{ deviceInfo.alarm_status || '-' }}</p>
          </div>
        </div>
        <div v-else class="tw-text-gray-400 tw-text-center tw-py-4">
          点击"查询设备信息"获取设备详细信息
        </div>
      </div>

      <!-- 状态信息（原有） -->
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

      <!-- 云台控制 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
        <h3 class="tw-text-lg tw-font-bold tw-mb-4 tw-border-b tw-pb-2">云台控制</h3>

        <!-- 通道选择提示 -->
        <el-alert
          v-if="channels.length === 0"
          title="该设备暂无通道，请先同步目录"
          type="info"
          :closable="false"
          show-icon
          class="tw-mb-4"
        />

        <el-alert
          v-else-if="!selectedChannelId"
          title="请先在下拉框中选择要控制的通道"
          type="warning"
          :closable="false"
          show-icon
          class="tw-mb-4"
        />

        <!-- 通道选择 -->
        <div class="tw-mb-4">
          <label class="tw-text-sm tw-font-medium tw-text-gray-700 tw-mb-2 tw-block">
            <el-icon class="tw-mr-1"><VideoCamera /></el-icon>
            选择通道
          </label>
          <el-select
            v-model="selectedChannelId"
            placeholder="请选择要控制的通道"
            style="width: 100%"
            :disabled="channels.length === 0"
          >
            <el-option
              v-for="channel in channels"
              :key="channel.channelId"
              :label="channel.name || channel.channelId"
              :value="channel.channelId"
            >
              <div class="tw-flex tw-justify-between tw-items-center">
                <span>{{ channel.name || channel.channelId }}</span>
                <el-tag size="small" :type="channel.status === '1' ? 'success' : 'info'" class="tw-ml-2">
                  {{ channel.status === '1' ? '在线' : '离线' }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
          <p class="tw-text-xs tw-text-gray-500 tw-mt-1" v-if="channels.length > 0">
            当前设备共 {{ channels.length }} 个通道
          </p>
        </div>

        <!-- PTZ 控制组件 -->
        <PTZControl
          :device-id="deviceId"
          :channel-id="selectedChannelId"
          :device-status="device?.status"
          :disabled="!selectedChannelId || device?.status !== '1'"
        />

        <!-- 测试按钮 -->
        <div class="tw-mt-4 tw-pt-4 tw-border-t">
          <el-button
            type="primary"
            :disabled="!selectedChannelId || device?.status !== '1' || testingPTZ"
            :loading="testingPTZ"
            @click="testPTZ"
          >
            <el-icon class="tw-mr-1"><Setting /></el-icon>
            测试云台
          </el-button>
          <p class="tw-text-xs tw-text-gray-500 tw-mt-2">
            点击测试按钮将发送一个简短的"向上"命令，验证云台控制是否正常
          </p>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
        <h3 class="tw-text-lg tw-font-bold tw-mb-4 tw-border-b tw-pb-2">操作</h3>
        <div class="tw-flex tw-gap-4">
          <el-button type="primary" @click="viewChannels">
            查看通道列表
          </el-button>
          <el-button type="success" @click="syncCatalog" :loading="syncing">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            同步目录
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
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Refresh, Search, Location, VideoCamera, Setting } from '@element-plus/icons-vue'
import { useDeviceStore, useChannelStore } from '@/stores/device'
import { deviceExtendApi, type DevicePosition, type DeviceStatusInfo, type DeviceInfoDetail } from '@/api/device-extend'
import { wsService } from '@/api/websocket'
import { ptzApi } from '@/api/ptz'
import PTZControl from '@/components/PTZControl.vue'

const route = useRoute()
const router = useRouter()
const deviceStore = useDeviceStore()
const channelStore = useChannelStore()

const deviceId = computed(() => route.params.deviceId as string)
const device = computed(() => deviceStore.currentDevice)

/** 通道列表 */
const channels = computed(() => channelStore.channels.filter(c => c.deviceId === deviceId.value))

/** 选中的通道ID */
const selectedChannelId = ref<string>('')

/** 设备位置信息 */
const position = ref<DevicePosition | null>(null)
const loadingPosition = ref(false)

/** 设备状态信息 */
const deviceStatus = ref<DeviceStatusInfo | null>(null)
const queryingStatus = ref(false)

/** 设备详细信息 */
const deviceInfo = ref<DeviceInfoDetail | null>(null)
const queryingInfo = ref(false)

/** 同步目录状态 */
const syncing = ref(false)

/** 测试云台状态 */
const testingPTZ = ref(false)

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

/** 同步目录 */
async function syncCatalog() {
  if (!deviceId.value) return
  syncing.value = true
  try {
    await deviceStore.syncCatalog(deviceId.value)
    ElMessage.success('同步请求已发送')
    // 刷新通道列表
    await channelStore.fetchChannels()
  } catch {
    ElMessage.error('同步失败')
  } finally {
    syncing.value = false
  }
}

/** 刷新位置信息 */
async function refreshPosition() {
  if (!deviceId.value) return
  loadingPosition.value = true
  try {
    const res = await deviceExtendApi.getPosition(deviceId.value)
    if (res.data) {
      position.value = res.data
    }
  } catch {
    // 静默处理
  } finally {
    loadingPosition.value = false
  }
}

/** 在地图中查看 */
function openMap() {
  if (!position.value) return
  const { longitude, latitude } = position.value
  // 使用高德地图或百度地图
  const url = `https://uri.amap.com/marker?position=${longitude},${latitude}&name=${device.value?.name || '设备位置'}`
  window.open(url, '_blank')
}

/** 查询设备实时状态 */
async function queryDeviceStatus() {
  if (!deviceId.value) return
  queryingStatus.value = true
  try {
    // 先发送查询请求
    await deviceExtendApi.queryStatus(deviceId.value)
    ElMessage.success('状态查询请求已发送，请稍后刷新查看结果')
    // 等待一段时间后获取状态
    setTimeout(async () => {
      try {
        const res = await deviceExtendApi.getStatus(deviceId.value)
        if (res.data) {
          deviceStatus.value = res.data
        }
      } catch {
        // 静默处理
      }
    }, 3000)
  } catch {
    ElMessage.error('查询失败')
  } finally {
    queryingStatus.value = false
  }
}

/** 查询设备信息 */
async function queryDeviceInfo() {
  if (!deviceId.value) return
  queryingInfo.value = true
  try {
    // 先发送查询请求
    await deviceExtendApi.queryInfo(deviceId.value)
    ElMessage.success('信息查询请求已发送，请稍后刷新查看结果')
    // 等待一段时间后获取信息
    setTimeout(async () => {
      try {
        const res = await deviceExtendApi.getInfo(deviceId.value)
        if (res.data) {
          deviceInfo.value = res.data
        }
      } catch {
        // 静默处理
      }
    }, 3000)
  } catch {
    ElMessage.error('查询失败')
  } finally {
    queryingInfo.value = false
  }
}

/** WebSocket 事件处理 */
let unsubscribeWS: (() => void) | null = null

function setupWebSocket() {
  unsubscribeWS = wsService.subscribe('device_position', (event) => {
    const data = event.data as DevicePosition
    if (data.device_id === deviceId.value) {
      position.value = data
    }
  })
}

/** 测试云台控制 */
async function testPTZ() {
  if (!deviceId.value || !selectedChannelId.value) {
    ElMessage.warning('请先选择通道')
    return
  }

  testingPTZ.value = true
  console.log('[测试云台] 开始测试:', {
    deviceId: deviceId.value,
    channelId: selectedChannelId.value
  })

  try {
    // 发送向上命令
    await ptzApi.control({
      device_id: deviceId.value,
      channel_id: selectedChannelId.value,
      direction: 'Up',
      speed: 128
    })
    console.log('[测试云台] 向上命令发送成功')

    // 等待 500ms
    await new Promise(resolve => setTimeout(resolve, 500))

    // 发送停止命令
    await ptzApi.control({
      device_id: deviceId.value,
      channel_id: selectedChannelId.value,
      direction: 'Stop',
      speed: 128
    })
    console.log('[测试云台] 停止命令发送成功')

    ElMessage.success('云台测试成功，请观察设备是否有响应')
  } catch (error) {
    console.error('[测试云台] 测试失败:', error)
    ElMessage.error('云台测试失败，请检查设备状态和网络连接')
  } finally {
    testingPTZ.value = false
  }
}

/** 监听通道列表变化，自动选择第一个通道 */
watch(channels, (newChannels) => {
  // 如果只有一个通道且未选择，自动选择
  if (newChannels.length === 1 && !selectedChannelId.value) {
    selectedChannelId.value = newChannels[0].channelId
    console.log('[自动选择] 只有一个通道，自动选中:', selectedChannelId.value)
  }
}, { immediate: true })

onMounted(() => {
  if (deviceId.value) {
    deviceStore.fetchDevice(deviceId.value)
    channelStore.fetchChannels()
    // 初始化数据
    refreshPosition()
    // 连接 WebSocket
    wsService.connect()
    setupWebSocket()
  }
})

onUnmounted(() => {
  if (unsubscribeWS) {
    unsubscribeWS()
  }
})
</script>