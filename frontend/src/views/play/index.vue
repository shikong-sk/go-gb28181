<template>
  <div class="tw-p-6">
    <!-- 页面标题 -->
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">视频播放</h1>
      <p class="tw-text-gray-500 tw-mt-1">实时视频预览</p>
    </div>

    <!-- 播放控制 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6 tw-mb-6">
      <el-form :inline="true" class="tw-flex tw-flex-wrap tw-gap-4">
        <el-form-item label="设备ID">
          <el-input
            v-model="deviceId"
            placeholder="请输入设备ID"
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item label="通道ID">
          <el-input
            v-model="channelId"
            placeholder="请输入通道ID"
            style="width: 200px"
          />
        </el-form-item>
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

    <!-- 视频播放器 -->
    <div class="tw-bg-black tw-rounded-lg tw-shadow tw-p-4 tw-mb-6" v-if="playing">
      <div class="tw-aspect-video tw-bg-gray-900 tw-rounded tw-overflow-hidden">
        <video
          ref="videoRef"
          class="tw-w-full tw-h-full"
          controls
          autoplay
          muted
        />
      </div>
      <div class="tw-mt-4 tw-text-white">
        <p class="tw-text-sm">
          <span class="tw-text-gray-400">流ID:</span> {{ currentStream?.stream_id }}
        </p>
        <p class="tw-text-sm tw-mt-1">
          <span class="tw-text-gray-400">播放地址:</span>
          <el-link :href="currentStream?.flv_url" target="_blank" type="primary" class="tw-ml-2">
            FLV
          </el-link>
          <el-link :href="currentStream?.hls_url" target="_blank" type="primary" class="tw-ml-2">
            HLS
          </el-link>
          <el-link :href="currentStream?.rtsp_url" target="_blank" type="primary" class="tw-ml-2">
            RTSP
          </el-link>
        </p>
      </div>
    </div>

    <!-- 播放会话列表 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-6">
      <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
        <h3 class="tw-text-lg tw-font-bold">当前播放会话</h3>
        <el-button size="small" @click="refreshSessions">
          <el-icon class="tw-mr-1"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
      <el-table :data="sessions" stripe style="width: 100%">
        <el-table-column prop="stream_id" label="流ID" min-width="180" />
        <el-table-column prop="device_id" label="设备ID" min-width="180" />
        <el-table-column prop="channel_id" label="通道ID" min-width="180" />
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 'playing' ? 'success' : 'info'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="开始时间" min-width="160" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button type="danger" size="small" link @click="stopSession(row.stream_id)">
              停止
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { VideoPlay, VideoPause, Refresh } from '@element-plus/icons-vue'
import { playApi, type PlayResponse, type SessionResponse } from '@/api/play'

// 状态
const deviceId = ref('')
const channelId = ref('')
const playing = ref(false)
const videoRef = ref<HTMLVideoElement | null>(null)
const currentStream = ref<PlayResponse | null>(null)
const sessions = ref<SessionResponse[]>([])

// FLV 播放器实例
let flvPlayer: unknown = null

/** 开始播放 */
async function startPlay() {
  if (!deviceId.value || !channelId.value) {
    ElMessage.warning('请输入设备ID和通道ID')
    return
  }

  try {
    playing.value = true
    const res = await playApi.start(deviceId.value, channelId.value)
    if (res.data) {
      currentStream.value = res.data
      ElMessage.success('播放请求已发送')
      // 播放 FLV 流
      playFlv(res.data.flv_url)
      // 刷新会话列表
      refreshSessions()
    }
  } catch {
    ElMessage.error('播放失败')
    playing.value = false
  }
}

/** 播放 FLV 流 */
async function playFlv(url: string) {
  if (!videoRef.value) return

  // 尝试使用 flv.js (如果已安装)
  try {
    // @ts-expect-error flvjs is optional
    if (typeof flvjs !== 'undefined') {
      // @ts-expect-error flvjs is optional
      flvPlayer = flvjs.createPlayer({
        type: 'flv',
        url: url,
        isLive: true,
      }, {
        enableWorker: true,
        enableStashBuffer: false,
      })
      flvPlayer.attachMediaElement(videoRef.value)
      flvPlayer.load()
      flvPlayer.play()
    } else {
      // 回退到原生播放 (HLS)
      videoRef.value.src = currentStream.value?.hls_url || url
      videoRef.value.play()
    }
  } catch {
    // 回退到原生播放
    if (videoRef.value) {
      videoRef.value.src = url
      videoRef.value.play()
    }
  }
}

/** 停止播放 */
async function stopPlay() {
  if (!currentStream.value) return

  try {
    await playApi.stop(currentStream.value.stream_id)
    ElMessage.success('停止成功')
  } catch {
    ElMessage.error('停止失败')
  } finally {
    // 销毁播放器
    if (flvPlayer) {
      // @ts-expect-error flvPlayer destroy
      if (flvPlayer.destroy) flvPlayer.destroy()
      flvPlayer = null
    }
    playing.value = false
    currentStream.value = null
    refreshSessions()
  }
}

/** 停止指定会话 */
async function stopSession(streamId: string) {
  try {
    await playApi.stop(streamId)
    ElMessage.success('停止成功')
    refreshSessions()
  } catch {
    ElMessage.error('停止失败')
  }
}

/** 刷新会话列表 */
async function refreshSessions() {
  try {
    const res = await playApi.getSessions()
    sessions.value = res.data || []
  } catch {
    console.error('获取会话列表失败')
  }
}

onMounted(() => {
  refreshSessions()
})

onUnmounted(() => {
  if (flvPlayer) {
    // @ts-expect-error flvPlayer destroy
    if (flvPlayer.destroy) flvPlayer.destroy()
  }
})
</script>