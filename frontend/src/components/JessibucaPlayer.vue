<template>
  <div ref="containerRef" class="jessibuca-container tw-w-full tw-h-full">
    <canvas ref="canvasRef" class="tw-w-full tw-h-full"></canvas>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
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
}

const props = withDefaults(defineProps<Props>(), {
  autoPlay: true,
  muted: true,
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

// 播放器实例
let conn: any = null
let demuxer: any = null
let videoDecoder: any = null
let audioDecoder: any = null
let renderer: any = null
let frameCount = 0

// 初始化播放器
async function initPlayer() {
  try {
    console.log('[Jessibuca] 开始初始化播放器')
    console.log('[Jessibuca] canvasRef:', canvasRef)
    console.log('[Jessibuca] canvasRef.value:', canvasRef?.value)

    // 等待DOM渲染完成
    await new Promise(resolve => setTimeout(resolve, 100))

    if (!canvasRef || !canvasRef.value) {
      const error = new Error('播放器canvas未找到，请检查DOM渲染')
      console.error('[Jessibuca]', error)
      throw error
    }

    console.log('[Jessibuca] Canvas元素已找到')

    const canvas = canvasRef.value
    const parent = canvas.parentElement
    if (parent) {
      canvas.width = parent.clientWidth || 640
      canvas.height = parent.clientHeight || 480
    } else {
      canvas.width = 640
      canvas.height = 480
    }

    console.log('[Jessibuca] Canvas尺寸:', canvas.width, 'x', canvas.height)

    // 创建渲染器
    renderer = new CanvasRenderer(canvas)

    // 创建视频解码器（SIMD版本性能更好）
    videoDecoder = new VideoDecoderSoftSIMD({
      yuvMode: false,
      canvas,
      workerMode: false,
      wasmPath: '/videodec_simd.wasm', // 从public目录加载
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
        emit('playing')
        console.log('播放成功')
      }
    })

    videoDecoder.on('error', (error: any) => {
      console.error('[VideoDecoder Error]', error)
      emit('error', new Error(error.errMsg || String(error)))
    })

    emit('ready')
    console.log('播放器初始化成功')

    // 自动播放
    if (props.autoPlay && props.url) {
      await play(props.url)
    }
  } catch (error) {
    console.error('播放器初始化失败:', error)
    emit('error', error instanceof Error ? error : new Error(String(error)))
  }
}

// 播放
async function play(url: string) {
  if (!videoDecoder || !audioDecoder) {
    const error = new Error('播放器未初始化')
    console.error('[Jessibuca]', error)
    emit('error', error)
    return
  }

  if (!url) {
    const error = new Error('播放URL为空')
    console.error('[Jessibuca]', error)
    emit('error', error)
    return
  }

  try {
    console.log('[Jessibuca] 开始播放:', url)
    frameCount = 0

    // 根据URL类型创建连接
    let urlType: string
    try {
      urlType = getURLType(url)
      console.log('[Jessibuca] URL类型:', urlType)
    } catch (e) {
      console.error('[Jessibuca] URL类型判断失败:', e)
      // 默认使用HTTP
      urlType = 'http'
    }

    if (urlType === 'ws' || urlType === 'wss') {
      conn = new WebSocketConnection(url)
    } else {
      conn = new HttpConnection(url)
    }

    // 创建FLV解复用器
    const mode = props.mode === 'live' ? DemuxMode.PUSH : DemuxMode.PULL
    demuxer = new FlvDemuxer(conn, mode)

    // 监听视频编码配置变化
    demuxer.on(DemuxEvent.VIDEO_ENCODER_CONFIG_CHANGED, (vconfig: any) => {
      console.log('[视频配置]', vconfig)
      videoDecoder.configure(vconfig)
    })

    // 监听音频编码配置变化
    demuxer.on(DemuxEvent.AUDIO_ENCODER_CONFIG_CHANGED, (aconfig: any) => {
      console.log('[音频配置]', aconfig)
      audioDecoder.configure(aconfig)
    })

    // 设置数据回调（PUSH模式用于实时流）
    if (mode === DemuxMode.PUSH) {
      demuxer.gotVideo = (data: any) => {
        if (videoDecoder && videoDecoder.config) {
          videoDecoder.decode(data)
        }
      }
      demuxer.gotAudio = (data: any) => {
        if (audioDecoder) {
          audioDecoder.decode(data)
        }
      }
      await conn.connect()
    } else {
      // PULL模式用于文件播放
      await conn.connect()
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
            if (audioDecoder) {
              audioDecoder.decode(chunk)
            }
          },
        })
      )
    }
  } catch (error) {
    console.error('播放失败:', error)
    emit('error', error instanceof Error ? error : new Error(String(error)))
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
</style>