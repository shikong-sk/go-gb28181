<template>
  <el-autocomplete
    v-model="inputValue"
    :fetch-suggestions="fetchSuggestions"
    :placeholder="placeholder"
    :clearable="clearable"
    :disabled="disabled"
    :style="{ width: width }"
    :debounce="300"
    value-key="channelId"
    @select="handleSelect"
    @clear="handleClear"
    @input="handleInput"
  >
    <template #default="{ item }">
      <div class="tw-flex tw-items-center tw-gap-2">
        <el-tag :type="item.status === '1' ? 'success' : 'info'" size="small">
          {{ item.status === '1' ? '在线' : '离线' }}
        </el-tag>
        <span class="tw-font-mono tw-text-sm">{{ item.channelId }}</span>
        <span class="tw-text-gray-500 tw-text-sm tw-ml-2 tw-truncate tw-max-w-32">{{ item.name }}</span>
      </div>
    </template>
    <template #loading>
      <div class="tw-flex tw-items-center tw-justify-center tw-py-2">
        <el-icon class="tw-animate-spin tw-mr-2"><Loading /></el-icon>
        <span class="tw-text-gray-500">搜索中...</span>
      </div>
    </template>
  </el-autocomplete>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { channelApi } from '@/api/device'
import type { Channel } from '@/types/device.d'
import { Loading } from '@element-plus/icons-vue'

// Props
const props = withDefaults(defineProps<{
  modelValue?: string
  deviceId?: string  // 可选：限定某个设备下的通道
  placeholder?: string
  clearable?: boolean
  disabled?: boolean
  width?: string
}>(), {
  modelValue: '',
  deviceId: '',
  placeholder: '输入搜索通道',
  clearable: true,
  disabled: false,
  width: '220px'
})

// Emits
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'select', channel: Channel): void
}>()

// 内部状态
const inputValue = ref(props.modelValue)
const loading = ref(false)

// 同步外部值
watch(() => props.modelValue, (val) => {
  inputValue.value = val
})

// 监听 deviceId 变化，清空输入
watch(() => props.deviceId, () => {
  // 如果 deviceId 变化且当前有值，清空（由父组件处理）
})

// 搜索建议 - 实时查询后端
async function fetchSuggestions(query: string, cb: (results: Channel[]) => void) {
  // 如果输入为空，返回空数组
  if (!query || query.trim() === '') {
    cb([])
    return
  }

  loading.value = true
  try {
    // 实时查询后端 API，使用 keyword 参数进行服务端搜索
    const response = await channelApi.getList({
      page: 1,
      pageSize: 20,
      deviceId: props.deviceId || undefined,
      keyword: query.trim(),
    })

    if (response.data?.list) {
      cb(response.data.list)
    } else {
      cb([])
    }
  } catch (error) {
    console.error('搜索通道失败:', error)
    cb([])
  } finally {
    loading.value = false
  }
}

// 选择通道
function handleSelect(item: Channel) {
  inputValue.value = item.channelId
  emit('update:modelValue', item.channelId)
  emit('change', item.channelId)
  emit('select', item)
}

// 清空
function handleClear() {
  emit('update:modelValue', '')
  emit('change', '')
}

// 输入变化
function handleInput(value: string) {
  emit('update:modelValue', value)
  emit('change', value)
}
</script>