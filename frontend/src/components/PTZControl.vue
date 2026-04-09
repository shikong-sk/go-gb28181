<template>
  <div class="ptz-control">
    <!-- 控制面板标题 -->
    <div class="tw-flex tw-justify-between tw-items-center tw-mb-4">
      <h4 class="tw-text-sm tw-font-bold tw-text-gray-700">云台控制</h4>
      <el-tag v-if="disabled" type="info" size="small">未选择通道</el-tag>
    </div>

    <!-- 禁用提示信息 -->
    <el-alert
      v-if="disabled"
      :title="disabledMessage"
      type="warning"
      :closable="false"
      show-icon
      class="tw-mb-4"
    />

    <!-- 速度控制 -->
    <div class="tw-mb-4">
      <label class="tw-text-xs tw-text-gray-500 tw-mb-1 tw-block">控制速度</label>
      <el-slider
        v-model="speed"
        :min="1"
        :max="255"
        :step="10"
        :disabled="disabled"
        show-input
        :show-input-controls="false"
        input-size="small"
      />
    </div>

    <!-- 方向控制面板 -->
    <div class="tw-flex tw-flex-col tw-gap-2 tw-mb-4">
      <!-- 第一行：上 -->
      <div class="tw-flex tw-justify-center">
        <el-button
          :disabled="disabled"
          @mousedown="startControl('Up')"
          @mouseup="stopControl"
          @mouseleave="stopControl"
          class="direction-btn"
        >
          <el-icon><ArrowUp /></el-icon>
        </el-button>
      </div>

      <!-- 第二行：左、停止、右 -->
      <div class="tw-flex tw-justify-center tw-gap-2">
        <el-button
          :disabled="disabled"
          @mousedown="startControl('Left')"
          @mouseup="stopControl"
          @mouseleave="stopControl"
          class="direction-btn"
        >
          <el-icon><ArrowLeft /></el-icon>
        </el-button>
        <el-button
          type="danger"
          :disabled="disabled"
          @click="stopControl"
          class="direction-btn"
          style="min-width: 60px;"
        >
          停止
        </el-button>
        <el-button
          :disabled="disabled"
          @mousedown="startControl('Right')"
          @mouseup="stopControl"
          @mouseleave="stopControl"
          class="direction-btn"
        >
          <el-icon><ArrowRight /></el-icon>
        </el-button>
      </div>

      <!-- 第三行：下 -->
      <div class="tw-flex tw-justify-center">
        <el-button
          :disabled="disabled"
          @mousedown="startControl('Down')"
          @mouseup="stopControl"
          @mouseleave="stopControl"
          class="direction-btn"
        >
          <el-icon><ArrowDown /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- 变焦控制 -->
    <div class="tw-flex tw-gap-2 tw-mb-4">
      <el-button
        :disabled="disabled"
        @mousedown="startControl('ZoomOut')"
        @mouseup="stopControl"
        @mouseleave="stopControl"
        class="tw-flex-1"
      >
        <el-icon class="tw-mr-1"><ZoomOut /></el-icon>
        缩小
      </el-button>
      <el-button
        :disabled="disabled"
        @mousedown="startControl('ZoomIn')"
        @mouseup="stopControl"
        @mouseleave="stopControl"
        class="tw-flex-1"
      >
        <el-icon class="tw-mr-1"><ZoomIn /></el-icon>
        放大
      </el-button>
    </div>

    <!-- 快速停止按钮 -->
    <el-button
      type="danger"
      :disabled="disabled"
      @click="emergencyStop"
      class="tw-w-full"
    >
      <el-icon class="tw-mr-1"><SwitchButton /></el-icon>
      紧急停止
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowUp, ArrowDown, ArrowLeft, ArrowRight, ZoomIn, ZoomOut, SwitchButton } from '@element-plus/icons-vue'
import { ptzApi, type PTZDirection } from '@/api/ptz'

/** 组件属性 */
const props = defineProps<{
  deviceId: string       // 设备ID
  channelId?: string     // 通道ID（可选）
  disabled?: boolean     // 是否禁用
  deviceStatus?: string  // 设备状态：'1' 在线，其他离线
}>()

/** 控制速度 */
const speed = ref(128)

/** 是否正在控制中 */
const controlling = ref(false)

/** 当前控制方向 */
const currentDirection = ref<PTZDirection | null>(null)

/** 禁用原因提示 */
const disabledMessage = computed(() => {
  if (!props.channelId) {
    return '请先选择通道'
  }
  if (props.deviceStatus !== '1') {
    return '设备离线，无法控制'
  }
  return '请先开始播放'
})

/** 开始云台控制 */
async function startControl(direction: PTZDirection) {
  console.log('[PTZ控制] 开始控制:', {
    deviceId: props.deviceId,
    channelId: props.channelId,
    direction,
    speed: speed.value,
    disabled: props.disabled
  })

  if (!props.deviceId || !props.channelId) {
    ElMessage.warning('请先选择设备和通道')
    return
  }

  controlling.value = true
  currentDirection.value = direction

  try {
    await ptzApi.control({
      device_id: props.deviceId,
      channel_id: props.channelId,
      direction,
      speed: speed.value,
    })
    console.log('[PTZ控制] 控制命令发送成功')
  } catch (error) {
    console.error('[PTZ控制] 控制失败:', error)
    ElMessage.error('云台控制失败')
    controlling.value = false
    currentDirection.value = null
  }
}

/** 停止云台控制 */
async function stopControl() {
  if (!controlling.value || !props.deviceId || !props.channelId) {
    return
  }

  console.log('[PTZ控制] 停止控制:', {
    deviceId: props.deviceId,
    channelId: props.channelId,
    direction: currentDirection.value
  })

  try {
    await ptzApi.control({
      device_id: props.deviceId,
      channel_id: props.channelId,
      direction: 'Stop',
      speed: speed.value,
    })
    console.log('[PTZ控制] 停止命令发送成功')
  } catch (error) {
    console.error('[PTZ控制] 停止失败:', error)
    // 静默处理停止失败
  } finally {
    controlling.value = false
    currentDirection.value = null
  }
}

/** 紧急停止 */
async function emergencyStop() {
  if (!props.deviceId || !props.channelId) {
    return
  }

  console.log('[PTZ控制] 紧急停止:', {
    deviceId: props.deviceId,
    channelId: props.channelId
  })

  try {
    await ptzApi.stop(props.deviceId, props.channelId)
    console.log('[PTZ控制] 紧急停止成功')
    ElMessage.success('已停止云台')
    controlling.value = false
    currentDirection.value = null
  } catch (error) {
    console.error('[PTZ控制] 紧急停止失败:', error)
    ElMessage.error('停止云台失败')
  }
}
</script>

<style scoped>
.ptz-control {
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
}

.direction-btn {
  min-width: 40px;
}
</style>