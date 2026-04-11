<template>
  <el-autocomplete
    v-model="inputValue"
    :fetch-suggestions="fetchSuggestions"
    :placeholder="placeholder"
    :clearable="clearable"
    :disabled="disabled"
    :style="{ width: width }"
    :debounce="300"
    value-key="deviceId"
    @select="handleSelect"
    @clear="handleClear"
    @input="handleInput"
  >
    <template #default="{ item }">
      <div class="tw-flex tw-items-center tw-gap-2">
        <el-tag :type="item.status === '1' ? 'success' : 'info'" size="small">
          {{ item.status === '1' ? '在线' : '离线' }}
        </el-tag>
        <span class="tw-font-mono tw-text-sm">{{ item.deviceId }}</span>
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
import { deviceApi } from '@/api/device'
import type { Device } from '@/types/device.d'
import { Loading } from '@element-plus/icons-vue'

// Props
const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  clearable?: boolean
  disabled?: boolean
  width?: string
}>(), {
  modelValue: '',
  placeholder: '输入搜索设备',
  clearable: true,
  disabled: false,
  width: '220px'
})

// Emits
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'select', device: Device): void
}>()

// 内部状态
const inputValue = ref(props.modelValue)
const loading = ref(false)
const lastQuery = ref('')

// 同步外部值
watch(() => props.modelValue, (val) => {
  inputValue.value = val
})

// 搜索建议 - 实时查询后端
async function fetchSuggestions(query: string, cb: (results: Device[]) => void) {
  lastQuery.value = query

  // 如果输入为空，返回空数组（el-autocomplete 会显示 placeholder）
  if (!query || query.trim() === '') {
    cb([])
    return
  }

  loading.value = true
  try {
    // 实时查询后端 API，使用 keyword 参数进行服务端搜索
    const response = await deviceApi.getList({
      page: 1,
      pageSize: 20,
      keyword: query.trim(),
    })

    if (response.data?.list) {
      cb(response.data.list)
    } else {
      cb([])
    }
  } catch (error) {
    console.error('搜索设备失败:', error)
    cb([])
  } finally {
    loading.value = false
  }
}

// 选择设备
function handleSelect(item: Device) {
  inputValue.value = item.deviceId
  emit('update:modelValue', item.deviceId)
  emit('change', item.deviceId)
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