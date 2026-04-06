<template>
  <div class="tw-p-6">
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">视频播放</h1>
      <p class="tw-text-gray-500 tw-mt-1">实时预览与历史录像回放</p>
    </div>

    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <el-radio-group v-model="playMode" class="tw-mb-4">
        <el-radio-button label="live">实时预览</el-radio-button>
        <el-radio-button label="playback">录像回放</el-radio-button>
      </el-radio-group>

      <!-- 录像回放模式提示 -->
      <el-alert
        v-if="playMode === 'playback'"
        type="info"
        title="录像回放需要选择时间范围"
        :closable="false"
        class="tw-mb-4"
      >
        <template #default>
          请在下方选择开始时间和结束时间，或先查询录像列表后点击播放按钮
        </template>
      </el-alert>

      <el-form :inline="true" class="tw-flex tw-flex-wrap tw-gap-4">
        <el-form-item label="设备ID">
          <el-input v-model="deviceId" placeholder="请输入设备ID" style="width: 220px" />
        </el-form-item>
        <el-form-item label="通道ID">
          <el-input v-model="channelId" placeholder="请输入通道ID" style="width: 220px" />
        </el-form-item>
        <template v-if="playMode === 'playback'">
          <el-form-item label="查询日期">
            <el-date-picker
              v-model="queryDate"
              type="date"
              placeholder="选择日期"
              format="YYYY-MM-DD"
              value-format="YYYY-MM-DD"
              style="width: 180px"
            />
          </el-form-item>
          <el-form-item label="开始时间">
            <el-date-picker
              v-model="playbackStartTime"
              type="datetime"
              placeholder="选择开始时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
              style="width: 220px"
            />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-date-picker
              v-model="playbackEndTime"
              type="datetime"
              placeholder="选择结束时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DD HH:mm:ss"
              style="width: 220px"
            />
          </el-form-item>
        </template>
        <el-form-item>
          <el-button type="primary" @click="startPlay" :loading="playing">
            <el-icon class="tw-mr-1"><VideoPlay /></el-icon>
            开始播放
          </el-button>
          <el-button type="danger" @click="stopPlay" :disabled="!playing">
            <el-icon class="tw-mr-1"><VideoPause /></el-icon>
            停止播放
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <div v-if="playMode === 'playback'" class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
        <h3 class="tw-text-lg tw-font-bold">录像查询</h3>
        <div class="tw-flex tw-gap-2">
          <el-button size="small" @click="queryRecords()" :loading="queryingRecords">
            <el-icon class="tw-mr-1"><Search /></el-icon>
            查询录像
          </el-button>
          <el-button size="small" type="primary" @click="fetchRecords()" :loading="fetchingRecords">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            拉取录像
          </el-button>
          <el-button size="small" type="warning" @click="queryRecords(true)" :loading="queryingRecords" :disabled="fetchingRecords">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            强制刷新
          </el-button>
        </div>
      </div>

      <!-- 缓存状态显示 -->
      <div v-if="recordCacheInfo" class="tw-mb-4 tw-flex tw-items-center tw-gap-4">
        <el-tag :type="getDataSourceType(recordCacheInfo.source)" size="large">
          {{ getDataSourceText(recordCacheInfo.source) }}
        </el-tag>
        <span v-if="recordCacheInfo.cached_at" class="tw-text-sm tw-text-gray-500">
          缓存时间: {{ recordCacheInfo.cached_at }}
        </span>
        <span v-if="recordCacheInfo.expires_in > 0" class="tw-text-sm tw-text-gray-500">
          {{ formatExpiresIn(recordCacheInfo.expires_in) }}
        </span>
        <span v-if="recordCacheInfo.total > 0" class="tw-text-sm tw-text-gray-600">
          共 {{ recordCacheInfo.total }} 条录像
        </span>
      </div>

      <!-- 拉取状态显示 -->
      <el-alert
        v-if="fetchStatus && fetchStatus.status !== 'completed'"
        :type="fetchStatus.status === 'failed' ? 'error' : fetchStatus.status === 'fetching' ? 'warning' : 'info'"
        :title="fetchStatus.message"
        :closable="false"
        class="tw-mb-4"
      />

      <el-table :data="recordList" stripe style="width: 100%" v-loading="queryingRecords || fetchingRecords">
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="start_time" label="开始时间" min-width="170" />
        <el-table-column prop="end_time" label="结束时间" min-width="170" />
        <el-table-column prop="file_size" label="文件大小" width="120">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="playRecord(row)">播放</el-button>
            <el-button type="success" size="small" link @click="downloadRecord(row)" class="tw-ml-2">下载</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 下载进度显示区域 -->
    <div v-if="downloading || downloadProgress" class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
        <h3 class="tw-text-lg tw-font-bold">录像下载</h3>
        <el-button size="small" type="danger" @click="cancelDownload" :disabled="!downloading">
          取消下载
        </el-button>
      </div>
      <el-card v-if="downloadProgress" shadow="never">
        <div class="tw-flex tw-items-center tw-mb-4">
          <el-tag :type="getDownloadStatusType(downloadProgress.status)" size="large">
            {{ getDownloadStatusText(downloadProgress.status) }}
          </el-tag>
          <span class="tw-ml-4 tw-text-gray-600">倍速: {{ downloadProgress.speed }}x</span>
        </div>
        <el-progress
          :percentage="downloadProgress.progress"
          :status="downloadProgress.status === 'completed' ? 'success' : downloadProgress.status === 'failed' ? 'exception' : undefined"
          :stroke-width="20"
          class="tw-mb-4"
        />
        <div class="tw-text-sm tw-text-gray-500">
          <p><span class="tw-font-medium">流ID:</span> {{ downloadProgress.stream_id }}</p>
          <p><span class="tw-font-medium">设备ID:</span> {{ downloadProgress.device_id }}</p>
          <p><span class="tw-font-medium">通道ID:</span> {{ downloadProgress.channel_id }}</p>
          <p><span class="tw-font-medium">时间范围:</span> {{ downloadProgress.start_time }} ~ {{ downloadProgress.end_time }}</p>
          <p v-if="downloadProgress.error" class="tw-text-red-500"><span class="tw-font-medium">错误:</span> {{ downloadProgress.error }}</p>
        </div>
      </el-card>
    </div>

    <!-- 播放区域与 PTZ 控制并排布局 -->
    <div v-if="playing" class="tw-grid tw-grid-cols-1 lg:tw-grid-cols-4 tw-gap-6 tw-mb-6">
      <!-- 视频播放区域 -->
      <div class="lg:tw-col-span-3 tw-bg-gray-900 tw-rounded-lg tw-shadow-lg tw-p-4 tw-border tw-border-gray-700">
        <div class="tw-aspect-video tw-bg-black tw-rounded tw-overflow-hidden tw-relative">
          <JessibucaPlayer
            v-if="currentStream?.flv_url"
            :url="currentStream.flv_url"
            :mode="currentStream.mode"
            @playing="onPlaying"
            @error="onPlayError"
            class="tw-w-full tw-h-full"
          />
          <div v-else class="tw-w-full tw-h-full tw-flex tw-items-center tw-justify-center tw-text-gray-400">
            <div class="tw-text-center">
              <el-icon size="48" class="tw-mb-4 tw-animate-pulse"><VideoPlay /></el-icon>
              <p>正在加载视频流...</p>
            </div>
          </div>
        </div>
        <!-- 播放信息 -->
        <div class="tw-mt-4 tw-text-white">
          <div class="tw-flex tw-items-center tw-gap-4 tw-mb-2">
            <el-tag size="small" :type="currentStream?.mode === 'live' ? 'success' : 'warning'" effect="dark">
              {{ currentStream?.mode === 'live' ? '实时预览' : '录像回放' }}
            </el-tag>
            <span class="tw-text-gray-400 tw-text-sm">{{ currentStream?.stream_id }}</span>
          </div>
          <p class="tw-text-sm tw-mt-1" v-if="currentSession?.playback_start">
            <el-icon class="tw-mr-1 tw-text-gray-400"><Clock /></el-icon>
            <span class="tw-text-gray-400">回放时间:</span>
            {{ currentSession?.playback_start }} ~ {{ currentSession?.playback_end }}
          </p>
          <div class="tw-flex tw-items-center tw-gap-2 tw-mt-2">
            <span class="tw-text-gray-400 tw-text-sm">播放地址:</span>
            <el-link :href="currentStream?.flv_url" target="_blank" type="primary" class="tw-text-sm">FLV</el-link>
            <el-link :href="currentStream?.hls_url" target="_blank" type="primary" class="tw-text-sm">HLS</el-link>
            <el-link :href="currentStream?.rtsp_url" target="_blank" type="primary" class="tw-text-sm">RTSP</el-link>
          </div>
        </div>
      </div>

      <!-- PTZ 云台控制面板 -->
      <div class="lg:tw-col-span-1 tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <PTZControl
          :device-id="deviceId"
          :channel-id="channelId"
          :disabled="!playing || playMode !== 'live'"
        />
        <div v-if="playMode !== 'live'" class="tw-mt-4 tw-text-center tw-text-gray-500 tw-text-sm">
          <el-icon class="tw-mr-1"><Warning /></el-icon>
          回放模式下不支持云台控制
        </div>
      </div>
    </div>

    <!-- 未播放时显示 PTZ 控制提示 -->
    <div v-if="!playing && deviceId && channelId" class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <div class="tw-text-center tw-text-gray-500">
        <p class="tw-mb-2">已选择设备和通道，点击"开始播放"后可使用云台控制</p>
        <PTZControl
          :device-id="deviceId"
          :channel-id="channelId"
          :disabled="true"
        />
      </div>
    </div>

    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
        <h3 class="tw-text-lg tw-font-bold">当前播放会话</h3>
        <el-button size="small" @click="refreshSessions">
          <el-icon class="tw-mr-1"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <el-table :data="sessions" stripe style="width: 100%">
        <el-table-column prop="stream_id" label="流ID" min-width="200" />
        <el-table-column prop="device_id" label="设备ID" min-width="180" />
        <el-table-column prop="channel_id" label="通道ID" min-width="180" />
        <el-table-column prop="mode" label="模式" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.mode === 'live' ? 'success' : 'warning'" size="small">
              {{ row.mode === 'live' ? '实时' : '回放' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'playing' ? 'success' : 'info'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="开始时间" min-width="170" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button type="danger" size="small" link @click="stopSession(row.stream_id)">停止</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Clock, Refresh, Search, VideoPause, VideoPlay, Warning } from '@element-plus/icons-vue'
import {
  playApi,
  type PlayMode,
  type PlayResponse,
  type RecordItem,
  type RecordListResponse,
  type SessionResponse,
  type FetchRecordsResponse,
} from '@/api/play'
import {
  downloadApi,
  type DownloadSessionResponse,
  type ProgressResponse,
} from '@/api/download'
import PTZControl from '@/components/PTZControl.vue'
import JessibucaPlayer from '@/components/JessibucaPlayer.vue'

const route = useRoute()

const playMode = ref<PlayMode>('live')
const deviceId = ref('')
const channelId = ref('')
const queryDate = ref('')
const playbackStartTime = ref('')
const playbackEndTime = ref('')
const playing = ref(false)
const queryingRecords = ref(false)
const currentStream = ref<PlayResponse | null>(null)
const currentSession = ref<SessionResponse | null>(null)
const sessions = ref<SessionResponse[]>([])
const recordList = ref<RecordItem[]>([])
const recordCacheInfo = ref<RecordListResponse | null>(null)

// 录像拉取相关状态
const fetchingRecords = ref(false)
const fetchStatus = ref<FetchRecordsResponse | null>(null)

// 下载相关状态
const downloading = ref(false)
const downloadSpeed = ref(1)
const currentDownload = ref<DownloadSessionResponse | null>(null)
const downloadProgress = ref<ProgressResponse | null>(null)
const downloadSessions = ref<DownloadSessionResponse[]>([])
let downloadProgressTimer: ReturnType<typeof setInterval> | null = null

function formatDateTime(date: Date): string {
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

/** 初始化回放时间（默认当天的前1小时） */
function initPlaybackTime() {
  const now = new Date()
  const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000)
  playbackStartTime.value = formatDateTime(oneHourAgo)
  playbackEndTime.value = formatDateTime(now)
}

function formatFileSize(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`
}

async function queryRecords(forceRefresh = false) {
  if (!deviceId.value || !channelId.value) {
    ElMessage.warning('请输入设备ID和通道ID')
    return
  }

  queryingRecords.value = true
  try {
    const res = await playApi.queryRecords({
      device_id: deviceId.value,
      channel_id: channelId.value,
      date: queryDate.value || undefined,
      force: forceRefresh,
    })
    recordList.value = res.data?.items || []
    recordCacheInfo.value = res.data || null

    // 根据数据来源显示不同提示
    if (res.data?.source === 'cache') {
      ElMessage.success(`使用缓存数据 (${res.data.item_count} 条录像)`)
    } else if (res.data?.source === 'device') {
      ElMessage.success(`实时查询完成 (${res.data.item_count} 条录像)`)
    }

    if (recordList.value.length === 0) {
      ElMessage.info('未查询到录像')
    }
  } catch {
    ElMessage.error('查询录像失败')
  } finally {
    queryingRecords.value = false
  }
}

/** 拉取录像（强制刷新缓存） */
async function fetchRecords() {
  if (!deviceId.value || !channelId.value) {
    ElMessage.warning('请输入设备ID和通道ID')
    return
  }

  fetchingRecords.value = true
  try {
    const res = await playApi.fetchRecords({
      device_id: deviceId.value,
      channel_id: channelId.value,
      date: queryDate.value || undefined,
    })
    fetchStatus.value = res.data || null

    if (res.data?.status === 'completed') {
      ElMessage.success(`录像拉取成功，共 ${res.data.item_count} 条`)
      // 拉取成功后重新查询
      await queryRecords()
    } else if (res.data?.status === 'failed') {
      ElMessage.error(`拉取失败: ${res.data.message}`)
    }
  } catch {
    ElMessage.error('拉取录像失败')
  } finally {
    fetchingRecords.value = false
  }
}

/** 获取拉取状态 */
async function getFetchStatus() {
  if (!deviceId.value || !channelId.value) return

  try {
    const res = await playApi.getFetchStatus(
      deviceId.value,
      channelId.value,
      queryDate.value || undefined
    )
    fetchStatus.value = res.data || null
  } catch {
    console.error('获取拉取状态失败')
  }
}

/** 获取数据来源标签类型 */
function getDataSourceType(source: string): 'success' | 'warning' | 'info' | '' {
  switch (source) {
    case 'cache':
      return 'success'
    case 'device':
      return 'warning'
    case 'db':
      return 'info'
    default:
      return ''
  }
}

/** 获取数据来源文本 */
function getDataSourceText(source: string): string {
  switch (source) {
    case 'cache':
      return '缓存数据'
    case 'device':
      return '实时查询'
    case 'db':
      return '数据库'
    default:
      return source
  }
}

/** 格式化缓存剩余时间 */
function formatExpiresIn(seconds: number): string {
  if (seconds <= 0) return '已过期'
  if (seconds < 60) return `${seconds}秒后过期`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟后过期`
  return `${Math.floor(seconds / 3600)}小时后过期`
}

async function playRecord(record: RecordItem) {
  playMode.value = 'playback'
  playbackStartTime.value = record.start_time
  playbackEndTime.value = record.end_time
  await startPlay()
}

async function startPlay() {
  if (!deviceId.value || !channelId.value) {
    ElMessage.warning('请输入设备ID和通道ID')
    return
  }

  // 录像回放模式下验证时间参数
  if (playMode.value === 'playback') {
    if (!playbackStartTime.value || !playbackEndTime.value) {
      ElMessage.warning('录像回放需要选择开始时间和结束时间')
      return
    }
  }

  try {
    playing.value = true
    const res = playMode.value === 'live'
      ? await playApi.start(deviceId.value, channelId.value)
      : await playApi.startPlayback({
          device_id: deviceId.value,
          channel_id: channelId.value,
          start_time: playbackStartTime.value,
          end_time: playbackEndTime.value,
        })

    if (res.data) {
      currentStream.value = res.data
      console.log('播放响应:', res.data)
      await refreshSessions()
      ElMessage.success('播放请求已发送')
    }
  } catch (error) {
    playing.value = false
    console.error('播放失败:', error)
    ElMessage.error('播放失败: ' + (error instanceof Error ? error.message : String(error)))
  }
}

function onPlaying() {
  console.log('播放开始')
}

function onPlayError(error: Error) {
  console.error('播放错误:', error)
  ElMessage.error('播放失败: ' + error.message)
  playing.value = false
}

async function stopPlay() {
  if (!currentStream.value) return

  try {
    await playApi.stop(currentStream.value.stream_id)
    ElMessage.success('停止成功')
  } catch {
    ElMessage.error('停止失败')
  } finally {
    playing.value = false
    currentStream.value = null
    currentSession.value = null
    await refreshSessions()
  }
}

async function stopSession(streamId: string) {
  try {
    await playApi.stop(streamId)
    if (currentStream.value?.stream_id === streamId) {
      await stopPlay()
      return
    }
    await refreshSessions()
    ElMessage.success('停止成功')
  } catch {
    ElMessage.error('停止失败')
  }
}

async function refreshSessions() {
  try {
    const res = await playApi.getSessions()
    sessions.value = res.data || []
    if (currentStream.value) {
      currentSession.value = sessions.value.find((item) => item.stream_id === currentStream.value?.stream_id) || null
    }
  } catch {
    console.error('获取会话列表失败')
  }
}

/** 开始录像下载 */
async function downloadRecord(record: RecordItem) {
  if (!deviceId.value || !channelId.value) {
    ElMessage.warning('请输入设备ID和通道ID')
    return
  }

  try {
    downloading.value = true
    const res = await downloadApi.startDownload({
      device_id: deviceId.value,
      channel_id: channelId.value,
      start_time: record.start_time,
      end_time: record.end_time,
      speed: downloadSpeed.value,
    })

    if (res.data) {
      currentDownload.value = res.data
      ElMessage.success('下载请求已发送')
      // 开始轮询下载进度
      startDownloadProgressPolling(res.data.stream_id)
    }
  } catch (error) {
    downloading.value = false
    console.error('开始下载失败:', error)
    ElMessage.error('开始下载失败: ' + (error instanceof Error ? error.message : String(error)))
  }
}

/** 开始轮询下载进度 */
function startDownloadProgressPolling(streamId: string) {
  // 清理之前的定时器
  if (downloadProgressTimer) {
    clearInterval(downloadProgressTimer)
  }

  // 立即获取一次进度
  fetchDownloadProgress(streamId)

  // 每2秒轮询进度
  downloadProgressTimer = setInterval(() => {
    fetchDownloadProgress(streamId)
  }, 2000)
}

/** 获取下载进度 */
async function fetchDownloadProgress(streamId: string) {
  try {
    const res = await downloadApi.getProgress(streamId)
    downloadProgress.value = res.data || null

    // 如果下载完成或失败，停止轮询
    if (res.data && (res.data.status === 'completed' || res.data.status === 'failed' || res.data.status === 'cancelled')) {
      stopDownloadProgressPolling()
      downloading.value = false

      if (res.data.status === 'completed') {
        ElMessage.success('录像下载完成')
      } else if (res.data.status === 'failed') {
        ElMessage.error('录像下载失败: ' + (res.data.error || '未知错误'))
      }
    }
  } catch (error) {
    console.error('获取下载进度失败:', error)
    // 如果获取失败，可能是会话已结束
    stopDownloadProgressPolling()
    downloading.value = false
  }
}

/** 停止轮询下载进度 */
function stopDownloadProgressPolling() {
  if (downloadProgressTimer) {
    clearInterval(downloadProgressTimer)
    downloadProgressTimer = null
  }
}

/** 取消下载 */
async function cancelDownload() {
  if (!currentDownload.value) return

  try {
    await downloadApi.cancelDownload(currentDownload.value.stream_id)
    stopDownloadProgressPolling()
    downloading.value = false
    downloadProgress.value = null
    currentDownload.value = null
    ElMessage.success('下载已取消')
  } catch (error) {
    console.error('取消下载失败:', error)
    ElMessage.error('取消下载失败')
  }
}

/** 获取下载状态类型（用于 el-tag） */
function getDownloadStatusType(status: string): 'success' | 'warning' | 'danger' | 'info' | '' {
  switch (status) {
    case 'completed':
      return 'success'
    case 'downloading':
      return 'warning'
    case 'failed':
      return 'danger'
    case 'cancelled':
      return 'info'
    default:
      return ''
  }
}

/** 获取下载状态文本 */
function getDownloadStatusText(status: string): string {
  switch (status) {
    case 'completed':
      return '下载完成'
    case 'downloading':
      return '下载中'
    case 'failed':
      return '下载失败'
    case 'cancelled':
      return '已取消'
    case 'pending':
      return '等待中'
    default:
      return status
  }
}

onMounted(() => {
  queryDate.value = formatDateTime(new Date()).slice(0, 10)
  // 初始化回放时间
  initPlaybackTime()

  // 从 URL 参数读取设备和通道ID
  const queryDeviceId = route.query.deviceId as string
  const queryChannelId = route.query.channelId as string
  if (queryDeviceId) {
    deviceId.value = queryDeviceId
  }
  if (queryChannelId) {
    channelId.value = queryChannelId
  }

  refreshSessions()
})

onUnmounted(() => {
  // 清理下载进度定时器
  stopDownloadProgressPolling()
})
</script>
