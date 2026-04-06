<template>
  <div ref="containerRef" class="jessibuca-container tw-w-full tw-h-full tw-relative">
    <!-- 加载状态显示 -->
    <div v-if="isLoading" class="loading-overlay tw-absolute tw-inset-0 tw-flex tw-items-center tw-justify-center tw-bg-gray-900 tw-z-10">
      <div class="tw-text-center">
        <div class="loading-spinner tw-mb-4"></div>
        <p class="tw-text-gray-400 tw-text-sm">{{ loadingText }}</p>
      </div>
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

    <!-- Canvas 播放区域 -->
    <canvas ref="canvasRef" class="tw-w-full tw-h-full"></canvas>

    <!-- 播放状态指示器 -->
    <div v-if="isPlaying && !hasError" class="playing-indicator tw-absolute tw-bottom-2 tw-right-2 tw-z-5">
      <el-tag type="success" size="small" effect="dark">
        <el-icon class="tw-mr-1 tw-animate-pulse"><VideoPlay /></el-icon>
        播放中
      </el-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { VideoCameraFilled, Refresh, VideoPlay } from '@element-plus/icons-vue'
import type { PlayMode } from '@/api/play'

// 使用预编译的 jv4 包
import { HttpConnection, WebSocketConnection, getURLType } from 'jv4-connection'
import { FlvDemuxer, DemuxEvent, DemuxMode } from 'jv4-demuxer'
import { VideoDecoderSoftSIMD, AudioDecoderSoft } from 'jv4-decoder'
import { CanvasRenderer } from 'jv4-renderer'

// Props 定义
interface Props {
  url: string
  mode: PlayMode
  autoPlay?: boolean
  muted?: boolean
  enableAudio?: boolean  // 是否启用音频解码（PCMA 解码器 WASM 有 bug，默认关闭）
}

const props = withDefaults(defineProps<Props>(), {
  autoPlay: true,
  muted: true,
  enableAudio: false,  // 默认关闭音频，因为 PCMA 解码器 WASM 有 bug
})

// Emits 定义
const emit = defineEmits<{
  (e: 'ready'): void
  (e: 'playing'): void
  (e: 'pause'): void
  (e: 'ended'): void
  (e: 'error', error: Error): void
}>()

const containerRef = ref<HTMLElement | null>(null)
const canvasRef = ref<HTMLCanvasElement | null>(null)

// 播放状态管理
const isLoading = ref(false)
const isPlaying = ref(false)
const hasError = ref(false)
const errorMessage = ref('')
const loadingText = ref('正在初始化播放器...')

// 播放器实例
let conn: any = null
let demuxer: any = null
let videoDecoder: any = null
let audioDecoder: any = null
let renderer: any = null
let frameCount = 0

// 初始化播放器
async function initPlayer() {
  isLoading.value = true
  loadingText.value = '正在初始化播放器...'
  hasError.value = false
  errorMessage.value = ''

  try {
    // 等待DOM渲染完成
    await new Promise(resolve => setTimeout(resolve, 100))

    if (!canvasRef || !canvasRef.value) {
      throw new Error('播放器canvas未找到，请检查DOM渲染')
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

    loadingText.value = '正在加载解码器...'

    // 创建渲染器
    renderer = new CanvasRenderer(canvas)

    // 创建视频解码器（SIMD版本性能更好）
    videoDecoder = new VideoDecoderSoftSIMD({
      yuvMode: false,
      canvas,
      workerMode: false,
      wasmPath: '/videodec_simd.wasm',
    })

    // 初始化视频解码器
    await videoDecoder.initialize({
      print: (text: string) => console.log('[VideoDecoder]', text),
      printErr: (text: string) => console.error('[VideoDecoder Error]', text),
      onAbort: () => console.error('[VideoDecoder] WASM aborted'),
    })

    // 创建音频解码器
    audioDecoder = new AudioDecoderSoft()
    await audioDecoder.initialize()

    // 监听视频帧事件
    videoDecoder.on('videoFrame', (videoFrame: VideoFrame | any) => {
      if (renderer) {
        renderer.writeVideo(videoFrame)
      }
      frameCount++
      if (frameCount === 5) {
        isLoading.value = false
        isPlaying.value = true
        emit('playing')
      }
    })

    videoDecoder.on('error', (error: any) => {
      console.error('[VideoDecoder Error]', error)
      hasError.value = true
      errorMessage.value = error.errMsg || '视频解码错误'
      isLoading.value = false
      isPlaying.value = false
      emit('error', new Error(error.errMsg || String(error)))
    })

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
  if (!videoDecoder || !audioDecoder) {
    hasError.value = true
    errorMessage.value = '播放器未初始化'
    emit('error', new Error('播放器未初始化'))
    return
  }

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

  try {
    frameCount = 0

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

    // 关键顺序：先创建demuxer（PUSH模式会设置conn.oput），再connect
    const mode = props.mode === 'live' ? DemuxMode.PUSH : DemuxMode.PULL
    // 重要：使用 'avcc' 格式而不是 'annexb'，避免 videoDecoderConfig 未设置时崩溃
    demuxer = new FlvDemuxer(conn, mode, 'avcc')

    // 监听视频编码配置变化
    demuxer.on(DemuxEvent.VIDEO_ENCODER_CONFIG_CHANGED, (vconfig: any) => {
      videoDecoder.configure(vconfig)
    })

    // 监听音频编码配置变化
    demuxer.on(DemuxEvent.AUDIO_ENCODER_CONFIG_CHANGED, (aconfig: any) => {
      audioDecoder.configure(aconfig)
    })

    // 监听解复用错误
    demuxer.on(DemuxEvent.DEMUX_ERROR, (error: Error) => {
      console.error('[Demuxer Error]', error)
      hasError.value = true
      errorMessage.value = error.message || '解复用错误'
      isLoading.value = false
      emit('error', error)
    })

    // 设置数据回调（PUSH模式用于实时流）
    if (mode === DemuxMode.PUSH) {
      demuxer.gotVideo = (data: any) => {
        if (videoDecoder && videoDecoder.config) {
          videoDecoder.decode(data)
        }
      }
      demuxer.gotAudio = (data: any) => {
        // PCMA 解码器 WASM 有 bug，默认禁用
        // 可通过 enableAudio prop 启用
        if (props.enableAudio && audioDecoder && audioDecoder.config) {
          try {
            audioDecoder.decode(data)
          } catch (err) {
            console.error('[AudioDecoder Error]', err)
          }
        }
      }
    }

    await conn.connect()

    // PULL模式额外处理
    if (mode === DemuxMode.PULL) {
      demuxer.videoReadable?.pipeTo(
        new WritableStream({
          write(chunk: any) {
            if (videoDecoder && videoDecoder.config) {
              videoDecoder.decode(chunk)
            }
          },
        })
      )
      demuxer.audioReadable?.pipeTo(
        new WritableStream({
          write(chunk: any) {
            // PCMA 解码器 WASM 有 bug，默认禁用
            if (props.enableAudio && audioDecoder && audioDecoder.config) {
              try {
                audioDecoder.decode(chunk)
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

// 重试播放
async function retryPlay() {
  if (props.url) {
    await destroy()
    await initPlayer()
  }
}

// 暂停
function pause() {
  if (conn) {
    conn.close()
    conn = null
    emit('pause')
  }
}

// 销毁播放器
async function destroy() {
  try {
    if (conn) {
      conn.close()
      conn = null
    }
    if (demuxer) {
      demuxer = null
    }
    if (videoDecoder) {
      videoDecoder.close?.()
      videoDecoder = null
    }
    if (audioDecoder) {
      audioDecoder.close?.()
      audioDecoder = null
    }
    if (renderer) {
      renderer.close?.()
      renderer = null
    }
    frameCount = 0
    isLoading.value = false
    isPlaying.value = false
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
})
</script>

<style scoped>
.jessibuca-container {
  background-color: #000;
  width: 100%;
  height: 100%;
  position: relative;
}

.jessibuca-container canvas {
  display: block;
  width: 100%;
  height: 100%;
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
.playing-indicator {
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
</style>