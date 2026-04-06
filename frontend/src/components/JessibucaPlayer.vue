<template>
  <div ref="containerRef" class="jessibuca-container tw-w-full tw-h-full tw-relative">
    <!-- 加载状态显示 -->
    <div v-if="isLoading" class="loading-overlay tw-absolute tw-inset-0 tw-flex tw-flex-col tw-items-center tw-justify-center tw-bg-gray-900 tw-z-10">
      <div class="loading-spinner"></div>
      <p class="tw-text-gray-400 tw-text-sm tw-mt-4">{{ loadingText }}</p>
    </div>

    <!-- 错误状态显示 -->
    <div v-if="hasError" class="error-overlay tw-absolute tw-inset-0 tw-flex tw-items-center tw-justify-center tw-bg-gray-900 tw-z-10">
      <div class="tw-text-center">
        <el-icon size="48" class="tw-text-red-500 tw-mb-4"><VideoCameraFilled /></el-icon>
        <p class="tw-text-red-400 tw-text-sm tw-mb-2">{{ errorMessage }}</p>
        <el-button type="primary" size="small" @click="retryPlay">
          <el-icon class="tw-mr-1"><Refresh /></el-icon>
          重试
        </el-button>
      </div>
    </div>

    <!-- Canvas 播放区域 (WASM/WebCodecs 模式) -->
    <canvas v-if="!useMSE" ref="canvasRef" class="tw-w-full tw-h-full"></canvas>

    <!-- Video 元素 (MSE 模式) -->
    <video v-if="useMSE" ref="videoRef" class="tw-w-full tw-h-full" muted playsinline></video>

    <!-- 播放状态指示器 -->
    <div v-if="isPlaying && !hasError && !showControls" class="playing-indicator tw-absolute tw-bottom-2 tw-right-2 tw-z-5">
      <el-tag type="success" size="small" effect="dark">
        <el-icon class="tw-mr-1 tw-animate-pulse"><VideoPlay /></el-icon>
        播放中
      </el-tag>
    </div>

    <!-- 控制条 -->
    <div v-if="showControls" class="controls-overlay tw-absolute tw-bottom-0 tw-left-0 tw-right-0 tw-z-10">
      <div class="controls-bg tw-flex tw-items-center tw-gap-2 tw-px-3 tw-py-2">
        <!-- 播放/暂停按钮 -->
        <el-button :icon="isPaused ? VideoPlay : VideoPause" circle size="small" @click="togglePause" />

        <!-- 当前时间 -->
        <span class="tw-text-white tw-text-xs tw-min-w-[50px]">{{ formatTime(currentTime) }}</span>

        <!-- 进度条 -->
        <div class="progress-bar tw-flex-1 tw-h-1 tw-bg-gray-600 tw-rounded tw-cursor-pointer tw-relative" @click="seekTo">
          <div class="progress-fill tw-h-full tw-bg-blue-500 tw-rounded" :style="{ width: progressPercent + '%' }"></div>
          <!-- 缓冲指示 -->
          <div v-if="bufferedPercent > 0" class="buffered-fill tw-absolute tw-top-0 tw-left-0 tw-h-full tw-bg-gray-400 tw-opacity-50 tw-rounded" :style="{ width: bufferedPercent + '%' }"></div>
        </div>

        <!-- 总时长 -->
        <span class="tw-text-white tw-text-xs tw-min-w-[50px]">{{ formatTime(duration) }}</span>

        <!-- 静音按钮 -->
        <el-button :icon="muted ? Mute : Microphone" circle size="small" @click="toggleMute" />

        <!-- 解码器切换下拉 -->
        <el-dropdown trigger="click" @command="switchDecoder">
          <el-button size="small" type="info">
            {{ getDecoderLabel() }}
            <el-icon class="tw-ml-1"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="wasm" :disabled="videoDecoder === 'wasm'">
                <el-icon class="tw-mr-1"><Cpu /></el-icon>WASM软解
              </el-dropdown-item>
              <el-dropdown-item command="webcodecs" :disabled="videoDecoder === 'webcodecs' || !webCodecsSupported">
                <el-icon class="tw-mr-1"><Monitor /></el-icon>WebCodecs硬解
                <span v-if="!webCodecsSupported" class="tw-text-gray-400 tw-ml-1">(不支持)</span>
              </el-dropdown-item>
              <el-dropdown-item command="mse" :disabled="videoDecoder === 'mse'">
                <el-icon class="tw-mr-1"><VideoPlay /></el-icon>MSE
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <!-- 旋转按钮 -->
        <el-button :icon="RefreshRight" circle size="small" @click="rotateVideo" />

        <!-- 页面内全屏按钮 -->
        <el-button :icon="isPageFullscreen ? Aim : Rank" circle size="small" @click="togglePageFullscreen" />

        <!-- 全屏按钮 -->
        <el-button :icon="FullScreen" circle size="small" @click="toggleFullscreen" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ArrowDown,
  Aim,
  Cpu,
  FullScreen,
  Monitor,
  Mute,
  Microphone,
  Rank,
  Refresh,
  RefreshRight,
  VideoCameraFilled,
  VideoPlay,
  VideoPause
} from '@element-plus/icons-vue'
import type { PlayMode } from '@/api/play'

// 使用预编译的 jv4 包
import { HttpConnection, WebSocketConnection, getURLType } from 'jv4-connection'
import { FlvDemuxer, DemuxEvent, DemuxMode } from 'jv4-demuxer'
import {
  VideoDecoderSoftSIMD,
  VideoDecoderHard,
  VideoDecoderMSE,
  AudioDecoderSoft,
  AudioDecoderHard
} from 'jv4-decoder'
import { CanvasRenderer } from 'jv4-renderer'

// 解码器类型
export type VideoDecoderType = 'wasm' | 'webcodecs' | 'mse'
export type AudioDecoderType = 'wasm' | 'webcodecs' | 'disabled'

// Props 定义
interface Props {
  url: string
  mode: PlayMode
  autoPlay?: boolean
  muted?: boolean
  enableAudio?: boolean  // 是否启用音频解码
  videoDecoder?: VideoDecoderType  // 视频解码器类型
  audioDecoder?: AudioDecoderType  // 音频解码器类型
  showControls?: boolean  // 是否显示控制条
  defaultDecoder?: VideoDecoderType  // 默认解码器（自动选择时可指定偏好）
}

const props = withDefaults(defineProps<Props>(), {
  autoPlay: true,
  muted: true,
  enableAudio: false,
  videoDecoder: 'auto',  // 默认自动选择最佳解码器
  audioDecoder: 'disabled',  // 默认禁用音频
  showControls: true,
  defaultDecoder: 'wasm',  // 默认偏好WASM（兼容性最好）
})

// Emits 定义
const emit = defineEmits<{
  (e: 'ready'): void
  (e: 'playing'): void
  (e: 'pause'): void
  (e: 'ended'): void
  (e: 'error', error: Error): void
  (e: 'timeupdate', time: number): void
  (e: 'decoderchange', decoder: VideoDecoderType): void
}>()

const containerRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)

// 播放状态管理
const isLoading = ref(false)
const isPlaying = ref(false)
const isPaused = ref(false)
const hasError = ref(false)
const errorMessage = ref('')
const loadingText = ref('正在初始化播放器...')
const currentTime = ref(0)
const duration = ref(0)
const muted = ref(props.muted)
const bufferedPercent = ref(0)
const rotation = ref(0) // 画面旋转角度：0, 90, 180, 270
const isPageFullscreen = ref(false) // 页面内全屏状态

// 断流重连状态
const isReconnecting = ref(false)
const reconnectCount = ref(0)
const reconnectMaxCount = 5  // 最大重连次数
const reconnectInterval = 10000  // 重连间隔（毫秒）
const userStopped = ref(false)  // 用户是否主动停止
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let lastFrameTime = 0  // 最后收到帧的时间
let frameCheckInterval: ReturnType<typeof setInterval> | null = null

// 解码器状态
const videoDecoder = ref<VideoDecoderType>(props.videoDecoder === 'auto' ? props.defaultDecoder : props.videoDecoder)
const webCodecsSupported = ref(false)

// 播放器实例
let conn: any = null
let demuxer: any = null
let videoDecoderInstance: any = null
let audioDecoderInstance: any = null
let renderer: any = null
let frameCount = 0
let statsInterval: ReturnType<typeof setInterval> | null = null

// 计算属性
const useMSE = computed(() => videoDecoder.value === 'mse')
const progressPercent = computed(() => duration.value > 0 ? (currentTime.value / duration.value) * 100 : 0)

// 检测 WebCodecs 支持
function checkWebCodecsSupport(): boolean {
  return typeof VideoDecoder !== 'undefined' && typeof VideoEncoder !== 'undefined'
}

// 获取解码器标签类型
function getDecoderTagType(): 'success' | 'warning' | 'info' | '' {
  switch (videoDecoder.value) {
    case 'wasm':
      return 'info'
    case 'webcodecs':
      return 'success'
    case 'mse':
      return 'warning'
    default:
      return ''
  }
}

// 获取解码器标签文本
function getDecoderLabel(): string {
  switch (videoDecoder.value) {
    case 'wasm':
      return 'WASM'
    case 'webcodecs':
      return '硬解'
    case 'mse':
      return 'MSE'
    default:
      return videoDecoder.value
  }
}

// 格式化时间
function formatTime(seconds: number): string {
  if (isNaN(seconds) || !isFinite(seconds) || seconds < 0) seconds = 0
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

// 创建视频解码器
function createVideoDecoderInstance(): any {
  switch (videoDecoder.value) {
    case 'webcodecs':
      if (!webCodecsSupported.value) {
        console.warn('[Jessibuca] WebCodecs不支持，回退到WASM')
        videoDecoder.value = 'wasm'
        return createVideoDecoderInstance()
      }
      return new VideoDecoderHard()
    case 'mse':
      return new VideoDecoderMSE()
    case 'wasm':
    default:
      return new VideoDecoderSoftSIMD({
        yuvMode: false,
        canvas: canvasRef.value!,
        workerMode: false,
        wasmPath: '/videodec_simd.wasm',
      })
  }
}

// 创建音频解码器
function createAudioDecoder(): any {
  if (!props.enableAudio && props.audioDecoder === 'disabled') {
    return null
  }

  switch (props.audioDecoder) {
    case 'webcodecs':
      if (!webCodecsSupported.value) {
        console.warn('[Jessibuca] WebCodecs音频不支持')
        return null
      }
      return new AudioDecoderHard()
    case 'wasm':
      // WASM音频解码器已知有bug，仅在显式启用时使用
      if (props.enableAudio) {
        console.warn('[Jessibuca] WASM音频解码器可能不稳定')
        return new AudioDecoderSoft()
      }
      return null
    case 'disabled':
    default:
      return null
  }
}

// 初始化播放器
async function initPlayer() {
  isLoading.value = true
  loadingText.value = '正在初始化播放器...'
  hasError.value = false
  errorMessage.value = ''

  // 检测 WebCodecs 支持
  webCodecsSupported.value = checkWebCodecsSupport()

  // 自动选择最佳解码器
  if (props.videoDecoder === 'auto') {
    if (webCodecsSupported.value) {
      videoDecoder.value = props.defaultDecoder === 'webcodecs' ? 'webcodecs' : 'wasm'
    } else {
      videoDecoder.value = 'wasm'
    }
  }

  try {
    // 等待DOM渲染完成
    await new Promise(resolve => setTimeout(resolve, 100))

    // MSE 模式初始化 video 元素
    if (useMSE.value) {
      if (!videoRef.value) {
        throw new Error('Video元素未找到')
      }
      videoDecoderInstance = createVideoDecoderInstance()
      await videoDecoderInstance.initialize(videoRef.value)
    } else {
      // WASM/WebCodecs 模式初始化 canvas
      if (!canvasRef.value) {
        throw new Error('Canvas元素未找到')
      }

      const canvas = canvasRef.value
      const parent = canvas.parentElement
      if (parent) {
        canvas.width = parent.clientWidth || 640
        canvas.height = parent.clientHeight || 480
      } else {
        canvas.width = 640
        canvas.height = 480
      }

      // 创建渲染器
      renderer = new CanvasRenderer(canvas)
      videoDecoderInstance = createVideoDecoderInstance()

      // WASM解码器需要额外初始化
      if (videoDecoder.value === 'wasm') {
        await videoDecoderInstance.initialize({
          print: (text: string) => console.log('[VideoDecoder]', text),
          printErr: (text: string) => console.error('[VideoDecoder Error]', text),
          onAbort: () => console.error('[VideoDecoder] WASM aborted'),
        })
      } else if (videoDecoder.value === 'webcodecs') {
        await videoDecoderInstance.initialize()
      }
    }

    loadingText.value = '正在加载解码器...'

    // 监听视频帧事件
    if (videoDecoderInstance) {
      videoDecoderInstance.on('videoFrame', (videoFrame: VideoFrame | any) => {
        if (renderer && !useMSE.value) {
          renderer.writeVideo(videoFrame)
        }
        frameCount++
        lastFrameTime = Date.now()  // 更新最后收到帧的时间
        if (frameCount === 5) {
          isLoading.value = false
          isPlaying.value = true
          emit('playing')
          startStatsUpdate()
          startFrameCheck()  // 启动断流检测
        }
      })

      videoDecoderInstance.on('error', (error: any) => {
        console.error('[VideoDecoder Error]', error)
        hasError.value = true
        errorMessage.value = error.errMsg || error.message || '视频解码错误'
        isLoading.value = false
        isPlaying.value = false
        emit('error', new Error(errorMessage.value))
      })
    }

    // 创建音频解码器
    audioDecoderInstance = createAudioDecoder()
    if (audioDecoderInstance) {
      await audioDecoderInstance.initialize()
    }

    isLoading.value = false
    emit('ready')

    // 自动播放
    if (props.autoPlay && props.url) {
      await play(props.url)
    }
  } catch (error) {
    console.error('播放器初始化失败:', error)
    hasError.value = true
    errorMessage.value = error instanceof Error ? error.message : '播放器初始化失败'
    isLoading.value = false
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
}

// 播放
async function play(url: string) {
  if (!url) {
    hasError.value = true
    errorMessage.value = '播放URL为空'
    emit('error', new Error('播放URL为空'))
    return
  }

  isLoading.value = true
  loadingText.value = '正在连接视频流...'
  hasError.value = false
  isPlaying.value = false
  isPaused.value = false
  frameCount = 0
  userStopped.value = false  // 重置用户停止标记
  isReconnecting.value = false
  lastFrameTime = Date.now()

  try {
    // 根据URL类型创建连接
    let urlType: string
    try {
      urlType = getURLType(url)
    } catch (e) {
      urlType = 'http'
    }

    if (urlType === 'ws' || urlType === 'wss') {
      conn = new WebSocketConnection(url)
    } else {
      conn = new HttpConnection(url)
    }

    // 创建 demuxer，使用 'avcc' 格式避免崩溃
    const mode = props.mode === 'live' ? DemuxMode.PUSH : DemuxMode.PULL
    demuxer = new FlvDemuxer(conn, mode, 'avcc')

    // 监听视频编码配置变化
    demuxer.on(DemuxEvent.VIDEO_ENCODER_CONFIG_CHANGED, (vconfig: any) => {
      console.log('[Jessibuca] 视频配置:', vconfig?.codec)
      if (videoDecoderInstance) {
        videoDecoderInstance.configure(vconfig)
      }
    })

    // 监听音频编码配置变化
    demuxer.on(DemuxEvent.AUDIO_ENCODER_CONFIG_CHANGED, (aconfig: any) => {
      console.log('[Jessibuca] 音频配置:', aconfig?.codec)
      if (audioDecoderInstance) {
        audioDecoderInstance.configure(aconfig)
      }
    })

    // 监听解复用错误
    demuxer.on(DemuxEvent.DEMUX_ERROR, (error: Error) => {
      console.error('[Demuxer Error]', error)
      isLoading.value = false
      isPlaying.value = false
      emit('error', error)

      // 如果不是用户主动停止，触发断流重连
      if (!userStopped.value) {
        console.warn('[Jessibuca] 解复用错误，触发断流重连')
        handleStreamDisconnect()
      } else {
        hasError.value = true
        errorMessage.value = error.message || '解复用错误'
      }
    })

    // 设置数据回调
    if (mode === DemuxMode.PUSH) {
      demuxer.gotVideo = (data: any) => {
        // 暂停时跳过解码
        if (isPaused.value) return
        if (videoDecoderInstance && videoDecoderInstance.config) {
          videoDecoderInstance.decode(data)
          // 更新时间
          if (data.timestamp !== undefined) {
            currentTime.value = data.timestamp / 1000
            emit('timeupdate', currentTime.value)
          }
        }
      }
      demuxer.gotAudio = (data: any) => {
        // 暂停时跳过解码
        if (isPaused.value) return
        if (audioDecoderInstance && audioDecoderInstance.config) {
          try {
            audioDecoderInstance.decode(data)
          } catch (err) {
            console.error('[AudioDecoder Error]', err)
          }
        }
      }
    }

    await conn.connect()

    // PULL模式处理
    if (mode === DemuxMode.PULL) {
      demuxer.videoReadable?.pipeTo(
        new WritableStream({
          write(chunk: any) {
            if (videoDecoderInstance && videoDecoderInstance.config) {
              videoDecoderInstance.decode(chunk)
            }
          },
        })
      )
      demuxer.audioReadable?.pipeTo(
        new WritableStream({
          write(chunk: any) {
            if (audioDecoderInstance && audioDecoderInstance.config) {
              try {
                audioDecoderInstance.decode(chunk)
              } catch (err) {
                console.error('[AudioDecoder Error]', err)
              }
            }
          },
        })
      )
    }
  } catch (error) {
    console.error('播放失败:', error)
    hasError.value = true
    errorMessage.value = error instanceof Error ? error.message : '视频流连接失败'
    isLoading.value = false
    isPlaying.value = false
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
}

// 开始统计更新
function startStatsUpdate() {
  if (statsInterval) {
    clearInterval(statsInterval)
  }

  statsInterval = setInterval(() => {
    if (isPlaying.value && !isPaused.value) {
      // MSE模式下从video元素获取缓冲信息
      if (useMSE.value && videoRef.value) {
        currentTime.value = videoRef.value.currentTime
        if (videoRef.value.buffered.length > 0) {
          const bufferedEnd = videoRef.value.buffered.end(videoRef.value.buffered.length - 1)
          bufferedPercent.value = duration.value > 0 ? (bufferedEnd / duration.value) * 100 : 0
        }
        emit('timeupdate', currentTime.value)
      }
    }
  }, 250)
}

// 启动断流检测
function startFrameCheck() {
  if (frameCheckInterval) {
    clearInterval(frameCheckInterval)
  }
  lastFrameTime = Date.now()
  userStopped.value = false

  // 每2秒检查一次是否有新帧
  frameCheckInterval = setInterval(() => {
    if (!isPlaying.value || isPaused.value || userStopped.value || isReconnecting.value) {
      return
    }

    const now = Date.now()
    const elapsed = now - lastFrameTime

    // 超过5秒没有收到新帧，判定为断流
    if (elapsed > 5000) {
      console.warn('[Jessibuca] 断流检测：超过5秒未收到视频帧，触发重连')
      handleStreamDisconnect()
    }
  }, 2000)
}

// 停止断流检测
function stopFrameCheck() {
  if (frameCheckInterval) {
    clearInterval(frameCheckInterval)
    frameCheckInterval = null
  }
}

// 处理断流
async function handleStreamDisconnect() {
  // 如果用户主动停止，不触发重连
  if (userStopped.value) {
    return
  }

  // 如果已在重连中，跳过
  if (isReconnecting.value) {
    return
  }

  // 检查重连次数
  if (reconnectCount.value >= reconnectMaxCount) {
    console.error('[Jessibuca] 已达最大重连次数，停止重连')
    hasError.value = true
    errorMessage.value = `播放中断，已重连${reconnectMaxCount}次失败`
    isReconnecting.value = false
    return
  }

  isReconnecting.value = true
  reconnectCount.value++
  console.log(`[Jessibuca] 开始第 ${reconnectCount.value} 次重连...`)

  // 显示重连状态
  hasError.value = false
  isLoading.value = true
  loadingText.value = `连接中断，正在重连 (${reconnectCount.value}/${reconnectMaxCount})...`

  try {
    // 关闭当前连接
    if (conn) {
      conn.close()
      conn = null
    }
    if (demuxer) {
      demuxer = null
    }

    // 等待一段时间后重连
    await new Promise(resolve => setTimeout(resolve, 1000))

    // 如果用户已停止，取消重连
    if (userStopped.value) {
      isReconnecting.value = false
      isLoading.value = false
      return
    }

    // 重新播放
    if (props.url) {
      await play(props.url)

      // 重连成功检查（等待一段时间确认是否有帧到达）
      await new Promise(resolve => setTimeout(resolve, 3000))

      if (isPlaying.value) {
        console.log('[Jessibuca] 重连成功')
        isReconnecting.value = false
        reconnectCount.value = 0
        lastFrameTime = Date.now()
      } else {
        // 重连后仍未播放成功，安排下次重连
        console.warn('[Jessibuca] 重连后仍未收到视频帧，安排下次重连')
        if (reconnectCount.value < reconnectMaxCount && !userStopped.value) {
          loadingText.value = `重连失败，${reconnectInterval / 1000}秒后重试 (${reconnectCount.value}/${reconnectMaxCount})`
          reconnectTimer = setTimeout(() => {
            isReconnecting.value = false  // 重置状态允许下次重连
            handleStreamDisconnect()
          }, reconnectInterval)
        } else {
          hasError.value = true
          errorMessage.value = `播放中断，已重连${reconnectMaxCount}次失败`
          isReconnecting.value = false
          isLoading.value = false
        }
      }
    }
  } catch (error) {
    console.error('[Jessibuca] 重连失败:', error)

    // 重连失败，安排下次重连
    if (reconnectCount.value < reconnectMaxCount && !userStopped.value) {
      console.log(`[Jessibuca] ${reconnectInterval / 1000}秒后进行下次重连...`)
      loadingText.value = `重连失败，${reconnectInterval / 1000}秒后重试 (${reconnectCount.value}/${reconnectMaxCount})`

      reconnectTimer = setTimeout(() => {
        isReconnecting.value = false  // 重置状态允许下次重连
        handleStreamDisconnect()
      }, reconnectInterval)
    } else {
      // 达到最大重连次数
      hasError.value = true
      errorMessage.value = `播放中断，已重连${reconnectMaxCount}次失败`
      isReconnecting.value = false
      isLoading.value = false
    }
  }
}

// 取消重连
function cancelReconnect() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  isReconnecting.value = false
}

// 重试播放
async function retryPlay() {
  if (props.url) {
    reconnectCount.value = 0  // 手动重试时重置重连计数
    userStopped.value = false
    await destroy()
    await initPlayer()
  }
}

// 暂停/继续
function togglePause() {
  if (isPaused.value) {
    // 恢复播放
    isPaused.value = false
    emit('playing')
    // MSE模式下恢复播放
    if (useMSE.value && videoRef.value) {
      videoRef.value.play()
    }
  } else {
    // 暂停
    isPaused.value = true
    emit('pause')
    // MSE模式下暂停
    if (useMSE.value && videoRef.value) {
      videoRef.value.pause()
    }
    // WASM/WebCodecs模式下，暂停时清空canvas保持最后一帧
    // 实际暂停由 gotVideo 回调中的 isPaused 检查实现
  }
}

// 静音切换
function toggleMute() {
  muted.value = !muted.value
  if (useMSE.value && videoRef.value) {
    videoRef.value.muted = muted.value
  }
}

// 跳转 (回放模式)
function seekTo(event: MouseEvent) {
  if (props.mode === 'live') {
    return // 直播模式不支持跳转
  }

  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  const percent = (event.clientX - rect.left) / rect.width
  const targetTime = percent * duration.value

  currentTime.value = targetTime

  // MSE模式下跳转
  if (useMSE.value && videoRef.value) {
    videoRef.value.currentTime = targetTime
  }

  emit('timeupdate', currentTime.value)
}

// 切换解码器
async function switchDecoder(decoderType: VideoDecoderType) {
  if (decoderType === videoDecoder.value) return

  if (decoderType === 'webcodecs' && !webCodecsSupported.value) {
    console.warn('[Jessibuca] WebCodecs不支持')
    return
  }

  const currentUrl = props.url
  const wasPlaying = isPlaying.value

  // 销毁当前播放器
  await destroy()

  // 切换解码器类型
  videoDecoder.value = decoderType
  emit('decoderchange', decoderType)

  // 重新初始化
  await initPlayer()

  // 如果之前在播放，恢复播放
  if (wasPlaying && currentUrl) {
    await play(currentUrl)
  }
}

// 画面旋转
function rotateVideo() {
  rotation.value = (rotation.value + 90) % 360
  applyRotation()
}

// 应用旋转
function applyRotation() {
  const target = useMSE.value ? videoRef.value : canvasRef.value
  if (target) {
    target.style.transform = `rotate(${rotation.value}deg)`
  }
}

// 全屏切换
function toggleFullscreen() {
  if (!containerRef.value) return

  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    containerRef.value.requestFullscreen()
  }
}

// 页面内全屏切换
function togglePageFullscreen() {
  isPageFullscreen.value = !isPageFullscreen.value
  if (containerRef.value) {
    if (isPageFullscreen.value) {
      containerRef.value.classList.add('page-fullscreen')
    } else {
      containerRef.value.classList.remove('page-fullscreen')
    }
  }
}

// 暂停
function pause() {
  userStopped.value = true  // 标记用户主动停止
  cancelReconnect()
  stopFrameCheck()
  if (conn) {
    conn.close()
    conn = null
    emit('pause')
  }
  isPaused.value = true
}

// 销毁播放器
async function destroy() {
  try {
    // 标记用户停止，取消重连
    userStopped.value = true
    cancelReconnect()
    stopFrameCheck()

    if (statsInterval) {
      clearInterval(statsInterval)
      statsInterval = null
    }

    if (conn) {
      conn.close()
      conn = null
    }
    if (demuxer) {
      demuxer = null
    }
    if (videoDecoderInstance) {
      videoDecoderInstance.close?.()
      videoDecoderInstance = null
    }
    if (audioDecoderInstance) {
      audioDecoderInstance.close?.()
      audioDecoderInstance = null
    }
    if (renderer) {
      renderer.close?.()
      renderer = null
    }
    frameCount = 0
    isLoading.value = false
    isPlaying.value = false
    isPaused.value = false
    currentTime.value = 0
    bufferedPercent.value = 0
    isReconnecting.value = false
    reconnectCount.value = 0
  } catch (error) {
    console.error('销毁播放器失败:', error)
  }
}

// 监听 URL 变化
watch(
  () => props.url,
  async (newUrl, oldUrl) => {
    if (newUrl && newUrl !== oldUrl) {
      await destroy()
      await play(newUrl)
    }
  }
)

// 监听解码器属性变化
watch(
  () => props.videoDecoder,
  async (newDecoder) => {
    if (newDecoder !== 'auto' && newDecoder !== videoDecoder.value) {
      await switchDecoder(newDecoder)
    }
  }
)

// 生命周期
onMounted(() => {
  initPlayer()
})

onBeforeUnmount(() => {
  destroy()
})

// 暴露方法给父组件
defineExpose({
  play,
  pause,
  destroy,
  switchDecoder,
  getDecoderType: () => videoDecoder.value,
  getCurrentTime: () => currentTime.value,
  isPlaying: () => isPlaying.value,
})
</script>

<style scoped>
.jessibuca-container {
  background-color: #000;
  width: 100%;
  height: 100%;
  position: relative;
}

/* 页面内全屏样式 */
.jessibuca-container.page-fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 9999;
}

.jessibuca-container canvas,
.jessibuca-container video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  transition: transform 0.3s ease;
}

/* 加载动画 */
.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(255, 255, 255, 0.2);
  border-top-color: #409EFF;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* 播放状态指示器动画 */
.playing-indicator,
.decoder-indicator {
  animation: fadeIn 0.3s ease-in;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

/* 加载和错误覆盖层过渡 */
.loading-overlay,
.error-overlay {
  animation: fadeIn 0.2s ease-in;
}

/* 控制条样式 */
.controls-overlay {
  animation: slideUp 0.3s ease-out;
}

@keyframes slideUp {
  from {
    transform: translateY(10px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.controls-bg {
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.7));
}

.progress-bar {
  min-width: 100px;
  position: relative;
}

.progress-fill {
  transition: width 0.1s ease;
}

.buffered-fill {
  pointer-events: none;
}
</style>