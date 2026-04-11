<template>
  <div class="tw-p-6">
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">录像下载</h1>
      <p class="tw-text-gray-500 tw-mt-1">从设备下载历史录像文件</p>
    </div>

    <!-- 新建下载任务 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <h3 class="tw-text-lg tw-font-bold tw-mb-4">新建下载任务</h3>
      <el-form :model="downloadForm" label-width="100px" class="tw-max-w-2xl">
        <el-form-item label="设备ID" required>
          <DeviceSelector v-model="downloadForm.device_id" placeholder="输入搜索设备" width="300px" @select="onDeviceSelect" />
        </el-form-item>
        <el-form-item label="通道ID" required>
          <ChannelSelector v-model="downloadForm.channel_id" :device-id="downloadForm.device_id" placeholder="输入搜索通道" width="300px" />
        </el-form-item>
        <el-form-item label="开始时间" required>
          <el-date-picker
            v-model="downloadForm.start_time"
            type="datetime"
            placeholder="选择录像开始时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 280px"
          />
        </el-form-item>
        <el-form-item label="结束时间" required>
          <el-date-picker
            v-model="downloadForm.end_time"
            type="datetime"
            placeholder="选择录像结束时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 280px"
          />
        </el-form-item>
        <el-form-item label="下载倍速">
          <el-select v-model="downloadForm.speed" style="width: 150px">
            <el-option label="1倍速 (正常)" :value="1" />
            <el-option label="2倍速" :value="2" />
            <el-option label="4倍速" :value="4" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="startDownload" :loading="starting">
            <el-icon class="tw-mr-1"><Download /></el-icon>
            开始下载
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 下载任务列表 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
        <h3 class="tw-text-lg tw-font-bold">下载任务列表</h3>
        <div class="tw-flex tw-items-center tw-gap-2">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索设备ID或通道ID"
            clearable
            style="width: 200px"
            @keyup.enter="filterSessions"
            @clear="filterSessions"
          >
            <template #append>
              <el-button @click="filterSessions">
                <el-icon><Search /></el-icon>
              </el-button>
            </template>
          </el-input>
          <el-button size="small" @click="refreshSessions" :loading="refreshing">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <el-table :data="sessions" stripe v-loading="refreshing">
        <el-table-column prop="stream_id" label="流ID" min-width="180" />
        <el-table-column prop="device_id" label="设备ID" min-width="180" />
        <el-table-column prop="channel_id" label="通道ID" min-width="180" />
        <el-table-column prop="start_time" label="录像开始时间" min-width="170" />
        <el-table-column prop="end_time" label="录像结束时间" min-width="170" />
        <el-table-column prop="speed" label="倍速" width="80">
          <template #default="{ row }">
            {{ row.speed }}x
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="progress" label="进度" width="120">
          <template #default="{ row }">
            <el-progress
              :percentage="row.progress"
              :status="row.status === 'completed' ? 'success' : row.status === 'error' ? 'exception' : undefined"
              :stroke-width="10"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'downloading'"
              type="danger"
              size="small"
              link
              @click="cancelDownload(row.stream_id)"
            >
              取消
            </el-button>
            <el-button
              v-if="row.status === 'error'"
              type="primary"
              size="small"
              link
              @click="retryDownload(row)"
            >
              重试
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="sessions.length === 0 && !refreshing" description="暂无下载任务" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, Refresh, Search } from '@element-plus/icons-vue'
import { downloadApi, type DownloadSessionResponse } from '@/api/download'
import DeviceSelector from '@/components/common/DeviceSelector.vue'
import ChannelSelector from '@/components/common/ChannelSelector.vue'
import type { Device } from '@/types/device.d'

// 下载表单
const downloadForm = ref({
  device_id: '',
  channel_id: '',
  start_time: '',
  end_time: '',
  speed: 1
})

// 状态
const starting = ref(false)
const refreshing = ref(false)
const sessions = ref<DownloadSessionResponse[]>([])
const allSessions = ref<DownloadSessionResponse[]>([])  // 存储所有会话用于搜索
const searchKeyword = ref('')

// 进度轮询定时器
let progressTimer: number | null = null

// 过滤会话列表
function filterSessions() {
  if (!searchKeyword.value) {
    sessions.value = allSessions.value
  } else {
    const keyword = searchKeyword.value.toLowerCase()
    sessions.value = allSessions.value.filter(s =>
      s.device_id.toLowerCase().includes(keyword) ||
      s.channel_id.toLowerCase().includes(keyword)
    )
  }
}

// 状态映射
function getStatusType(status: string): '' | 'success' | 'warning' | 'info' | 'danger' {
  const map: Record<string, '' | 'success' | 'warning' | 'info' | 'danger'> = {
    pending: 'info',
    downloading: 'warning',
    completed: 'success',
    cancelled: 'info',
    error: 'danger'
  }
  return map[status] || 'info'
}

function getStatusText(status: string): string {
  const map: Record<string, string> = {
    pending: '等待中',
    downloading: '下载中',
    completed: '已完成',
    cancelled: '已取消',
    error: '失败'
  }
  return map[status] || status
}

// 设备选择后清空通道
function onDeviceSelect(device: Device) {
  // 切换设备时清空通道选择
  downloadForm.value.channel_id = ''
}

// 开始下载
async function startDownload() {
  // 表单验证
  if (!downloadForm.value.device_id) {
    ElMessage.warning('请输入设备ID')
    return
  }
  if (!downloadForm.value.channel_id) {
    ElMessage.warning('请输入通道ID')
    return
  }
  if (!downloadForm.value.start_time) {
    ElMessage.warning('请选择开始时间')
    return
  }
  if (!downloadForm.value.end_time) {
    ElMessage.warning('请选择结束时间')
    return
  }

  starting.value = true
  try {
    const response = await downloadApi.startDownload({
      device_id: downloadForm.value.device_id,
      channel_id: downloadForm.value.channel_id,
      start_time: downloadForm.value.start_time,
      end_time: downloadForm.value.end_time,
      speed: downloadForm.value.speed
    })

    if (response.data) {
      ElMessage.success('下载任务已创建')
      // 刷新列表
      await refreshSessions()
      // 清空表单
      downloadForm.value = {
        device_id: '',
        channel_id: '',
        start_time: '',
        end_time: '',
        speed: 1
      }
    }
  } catch (error: any) {
    ElMessage.error(error.message || '创建下载任务失败')
  } finally {
    starting.value = false
  }
}

// 刷新下载任务列表
async function refreshSessions() {
  refreshing.value = true
  try {
    const response = await downloadApi.getSessions()
    if (response.data) {
      allSessions.value = response.data
      filterSessions()  // 应用搜索过滤
    }
  } catch (error: any) {
    ElMessage.error(error.message || '获取下载任务列表失败')
  } finally {
    refreshing.value = false
  }
}

// 取消下载
async function cancelDownload(streamId: string) {
  try {
    await ElMessageBox.confirm('确定要取消该下载任务吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    await downloadApi.cancelDownload(streamId)
    ElMessage.success('下载任务已取消')
    await refreshSessions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '取消下载失败')
    }
  }
}

// 重试下载
async function retryDownload(session: DownloadSessionResponse) {
  downloadForm.value = {
    device_id: session.device_id,
    channel_id: session.channel_id,
    start_time: session.start_time,
    end_time: session.end_time,
    speed: session.speed || 1
  }
  ElMessage.info('已填充表单，请点击"开始下载"重试')
}

// 更新进行中任务的进度
async function updateProgress() {
  const downloadingSessions = sessions.value.filter(s => s.status === 'downloading')
  if (downloadingSessions.length === 0) return

  for (const session of downloadingSessions) {
    try {
      const response = await downloadApi.getProgress(session.stream_id)
      if (response.data) {
        // 更新对应会话的状态
        const index = sessions.value.findIndex(s => s.stream_id === session.stream_id)
        if (index >= 0) {
          sessions.value[index] = response.data
        }
      }
    } catch (error) {
      // 忽略单个进度查询错误
    }
  }
}

// 启动进度轮询
function startProgressPolling() {
  if (progressTimer) return
  progressTimer = window.setInterval(updateProgress, 2000)
}

// 停止进度轮询
function stopProgressPolling() {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

// 生命周期
onMounted(() => {
  refreshSessions()
  startProgressPolling()
})

onUnmounted(() => {
  stopProgressPolling()
})
</script>