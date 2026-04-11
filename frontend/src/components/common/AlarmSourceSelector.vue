<template>
  <el-autocomplete
    v-model="inputValue"
    :fetch-suggestions="fetchSuggestions"
    :placeholder="placeholder"
    :clearable="clearable"
    :disabled="disabled"
    :style="{ width: width }"
    :debounce="300"
    @select="handleSelect"
    @clear="handleClear"
    @input="handleInput"
  >
    <template #default="{ item }">
      <div class="tw-flex tw-items-center tw-gap-2">
        <el-tag :type="item._type === 'device' ? 'primary' : 'success'" size="small">
          {{ item._type === 'device' ? '设备' : '通道' }}
        </el-tag>
        <el-tag :type="item.status === '1' ? 'success' : 'info'" size="small">
          {{ item.status === '1' ? '在线' : '离线' }}
        </el-tag>
        <span class="tw-font-mono tw-text-sm">{{ item.id }}</span>
        <span class="tw-text-gray-500 tw-text-sm tw-ml-2 tw-truncate tw-max-w-24">{{ item.name || '-' }}</span>
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
import { ref, watch } from 'vue'
import { deviceApi, channelApi } from '@/api/device'
import { Loading } from '@element-plus/icons-vue'

interface SuggestionItem {
  id: string
  name: string
  status: string
  _type: 'device' | 'channel'
}

// Props
const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  clearable?: boolean
  disabled?: boolean
  width?: string
}>(), {
  modelValue: '',
  placeholder: '输入搜索设备/通道',
  clearable: true,
  disabled: false,
  width: '220px'
})

// Emits
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
}>()

// 内部状态
const inputValue = ref(props.modelValue)
const loading = ref(false)

// 同步外部值
watch(() => props.modelValue, (val) => {
  inputValue.value = val
})

// 搜索建议 - 同时搜索设备和通道
async function fetchSuggestions(query: string, cb: (results: SuggestionItem[]) => void) {
  // 如果输入为空，返回空数组
  if (!query || query.trim() === '') {
    cb([])
    return
  }

  loading.value = true
  try {
    // 并行搜索设备和通道
    const [deviceRes, channelRes] = await Promise.all([
      deviceApi.getList({ page: 1, pageSize: 10, keyword: query.trim() }),
      channelApi.getList({ page: 1, pageSize: 10, keyword: query.trim() })
    ])

    const results: SuggestionItem[] = []

    // 添加设备结果
    if (deviceRes.data?.list) {
      deviceRes.data.list.forEach(device => {
        results.push({
          id: device.deviceId,
          name: device.name || '',
          status: device.status,
          _type: 'device'
        })
      })
    }

    // 添加通道结果
    if (channelRes.data?.list) {
      channelRes.data.list.forEach(channel => {
        results.push({
          id: channel.channelId,
          name: channel.name || '',
          status: channel.status,
          _type: 'channel'
        })
      })
    }

    cb(results.slice(0, 20))
  } catch (error) {
    console.error('搜索报警源失败:', error)
    cb([])
  } finally {
    loading.value = false
  }
}

// 选择
function handleSelect(item: SuggestionItem) {
  inputValue.value = item.id
  emit('update:modelValue', item.id)
  emit('change', item.id)
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
