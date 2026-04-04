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
        <el-button size="small" @click="queryRecords" :loading="queryingRecords">
          <el-icon class="tw-mr-1"><Search /></el-icon>
          查询录像
        </el-button>
      </div>
      <el-table :data="recordList" stripe style="width: 100%" v-loading="queryingRecords">
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="start_time" label="开始时间" min-width="170" />
        <el-table-column prop="end_time" label="结束时间" min-width="170" />
        <el-table-column prop="file_size" label="文件大小" width="120">
          <template #default="{ row }">
            {{ formatFileSize(row.file_size) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="playRecord(row)">播放</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 播放区域与 PTZ 控制并排布局 -->
    <div v-if="playing" class="tw-grid tw-grid-cols-1 lg:tw-grid-cols-4 tw-gap-6 tw-mb-6">
      <!-- 视频播放区域 -->
      <div class="lg:tw-col-span-3 tw-bg-black tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-aspect-video tw-bg-gray-900 tw-rounded tw-overflow-hidden">
          <video ref="videoRef" class="tw-w-full tw-h-full" controls autoplay muted />
        </div>
        <div class="tw-mt-4 tw-text-white">
          <p class="tw-text-sm"><span class="tw-text-gray-400">流ID:</span> {{ currentStream?.stream_id }}</p>
          <p class="tw-text-sm tw-mt-1">
            <span class="tw-text-gray-400">播放模式:</span>
            <el-tag size="small" :type="currentStream?.mode === 'live' ? 'success' : 'warning'" class="tw-ml-1">
              {{ currentStream?.mode === 'live' ? '实时' : '回放' }}
            </el-tag>
          </p>
          <p class="tw-text-sm tw-mt-1" v-if="currentSession?.playback_start">
            <span class="tw-text-gray-400">回放时间:</span>
            {{ currentSession?.playback_start }} ~ {{ currentSession?.playback_end }}
          </p>
          <p class="tw-text-sm tw-mt-1">
            <span class="tw-text-gray-400">播放地址:</span>
            <el-link :href="currentStream?.flv_url" target="_blank" type="primary" class="tw-ml-2">FLV</el-link>
            <el-link :href="currentStream?.hls_url" target="_blank" type="primary" class="tw-ml-2">HLS</el-link>
            <el-link :href="currentStream?.rtsp_url" target="_blank" type="primary" class="tw-ml-2">RTSP</el-link>
          </p>
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
import { Refresh, Search, VideoPause, VideoPlay, Warning } from '@element-plus/icons-vue'
import {
  playApi,
  type PlayMode,
  type PlayResponse,
  type RecordItem,
  type SessionResponse,
} from '@/api/play'
import PTZControl from '@/components/PTZControl.vue'

const route = useRoute()

const playMode = ref<PlayMode>('live')
const deviceId = ref('')
const channelId = ref('')
const queryDate = ref('')
const playbackStartTime = ref('')
const playbackEndTime = ref('')
const playing = ref(false)
const queryingRecords = ref(false)
const videoRef = ref<HTMLVideoElement | null>(null)
const currentStream = ref<PlayResponse | null>(null)
const currentSession = ref<SessionResponse | null>(null)
const sessions = ref<SessionResponse[]>([])
const recordList = ref<RecordItem[]>([])

let flvPlayer: unknown = null

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

async function queryRecords() {
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
    })
    recordList.value = res.data?.items || []
    if (recordList.value.length === 0) {
      ElMessage.info('未查询到录像')
    }
  } catch {
    ElMessage.error('查询录像失败')
  } finally {
    queryingRecords.value = false
  }
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
      console.log('FLV URL:', res.data.flv_url)
      console.log('HLS URL:', res.data.hls_url)
      await playFlv(res.data.flv_url)
      await refreshSessions()
      ElMessage.success('播放请求已发送')
    }
  } catch (error) {
    playing.value = false
    console.error('播放失败:', error)
    ElMessage.error('播放失败: ' + (error instanceof Error ? error.message : String(error)))
  }
}

async function playFlv(url: string) {
  if (!videoRef.value) {
    console.error('视频元素未找到')
    return
  }

  try {
    // @ts-expect-error flvjs is optional
    if (typeof flvjs !== 'undefined' && flvjs.isSupported()) {
      console.log('使用 flv.js 播放:', url)
      // @ts-expect-error flvjs is optional
      flvPlayer = flvjs.createPlayer(
        {
          type: 'flv',
          url,
          isLive: currentStream.value?.mode === 'live',
        },
        {
          enableWorker: false, // 禁用 worker 避免打包问题
          enableStashBuffer: false,
          lazyLoad: false,
        }
      )
      // @ts-expect-error flvjs is optional
      flvPlayer.attachMediaElement(videoRef.value)
      // @ts-expect-error flvjs is optional
      flvPlayer.load()
      // @ts-expect-error flvjs is optional
      await flvPlayer.play()
      console.log('flv.js 播放成功')
      return
    } else {
      console.warn('flv.js 未加载或不支持，尝试直接播放')
    }
  } catch (error) {
    console.error('flv.js 播放失败:', error)
  }

  // 降级到 HLS 或直接播放
  const fallbackUrl = currentStream.value?.hls_url || url
  console.log('降级播放:', fallbackUrl)
  try {
    videoRef.value.src = fallbackUrl
    await videoRef.value.play()
    console.log('降级播放成功')
  } catch (error) {
    console.error('降级播放失败:', error)
    throw new Error('无法播放视频流，请检查流媒体地址是否正确')
  }
}

async function stopPlay() {
  if (!currentStream.value) return

  try {
    await playApi.stop(currentStream.value.stream_id)
    ElMessage.success('停止成功')
  } catch {
    ElMessage.error('停止失败')
  } finally {
    if (flvPlayer) {
      // @ts-expect-error flvjs is optional
      flvPlayer.destroy?.()
      flvPlayer = null
    }
    playing.value = false
    currentStream.value = null
    currentSession.value = null
    if (videoRef.value) {
      videoRef.value.pause()
      videoRef.value.removeAttribute('src')
      videoRef.value.load()
    }
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
  if (flvPlayer) {
    // @ts-expect-error flvjs is optional
    flvPlayer.destroy?.()
  }
})
</script>
