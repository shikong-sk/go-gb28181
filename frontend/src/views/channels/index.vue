<template>
  <div class="tw-p-6">
    <!-- 页面标题 -->
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">通道管理</h1>
      <p class="tw-text-gray-500 tw-mt-1">管理 GB28181 设备通道</p>
    </div>

    <!-- 筛选栏 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4 tw-mb-6">
      <el-form :inline="true" class="tw-flex tw-flex-wrap tw-gap-4">
        <el-form-item label="设备ID">
          <el-input
            v-model="deviceIdFilter"
            placeholder="请输入设备ID"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item label="通道状态">
          <el-select
            v-model="statusFilter"
            placeholder="全部"
            clearable
            style="width: 120px"
            @change="handleFilterChange"
          >
            <el-option label="在线" value="1" />
            <el-option label="离线" value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleRefresh">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            刷新
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 统计卡片 -->
    <div class="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4 tw-mb-6" v-if="channelStore.stats.total > 0">
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-purple-100 tw-text-purple-600">
            <el-icon size="24"><VideoCamera /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">通道总数</p>
            <p class="tw-text-2xl tw-font-bold tw-text-gray-800">{{ channelStore.stats.total }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-green-100 tw-text-green-600">
            <el-icon size="24"><CircleCheck /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">在线通道</p>
            <p class="tw-text-2xl tw-font-bold tw-text-green-600">{{ channelStore.stats.online }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-red-100 tw-text-red-600">
            <el-icon size="24"><CircleClose /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">离线通道</p>
            <p class="tw-text-2xl tw-font-bold tw-text-red-600">{{ channelStore.stats.offline }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 通道列表 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow">
      <el-table
        v-loading="channelStore.loading"
        :data="channelStore.channels"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="channelId" label="通道ID" min-width="180" />
        <el-table-column prop="name" label="通道名称" min-width="150">
          <template #default="{ row }">
            {{ row.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="deviceId" label="所属设备" min-width="180" />
        <el-table-column prop="manufacturer" label="厂商" min-width="120">
          <template #default="{ row }">
            {{ row.manufacturer || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === '1' ? 'success' : 'danger'" size="small">
              {{ row.status === '1' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="ptzType" label="PTZ类型" width="100" align="center">
          <template #default="{ row }">
            {{ getPTZTypeName(row.ptzType) }}
          </template>
        </el-table-column>
        <el-table-column prop="hasAudio" label="音频" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.hasAudio" type="success" size="small">有</el-tag>
            <el-tag v-else type="info" size="small">无</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="handlePlay(row)">
              播放
            </el-button>
            <el-popconfirm
              title="确定删除该通道？"
              confirm-button-text="确定"
              cancel-button-text="取消"
              @confirm="handleDelete(row.channelId, row.deviceId)"
            >
              <template #reference>
                <el-button type="danger" size="small" link>删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="tw-flex tw-justify-end tw-p-4">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="channelStore.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { VideoCamera, CircleCheck, CircleClose, Refresh } from '@element-plus/icons-vue'
import { useChannelStore } from '@/stores/device'
import type { Channel } from '@/types/device'

const route = useRoute()
const router = useRouter()
const channelStore = useChannelStore()

const deviceIdFilter = ref<string>('')
const statusFilter = ref<string>('')
const currentPage = ref(1)
const pageSize = ref(20)

// PTZ 类型名称
const ptzTypeNames: Record<number, string> = {
  0: '未知',
  1: '球机',
  2: '半球',
  3: '固定',
  4: '遥控',
}

function getPTZTypeName(type: number): string {
  return ptzTypeNames[type] || '未知'
}

/** 刷新列表 */
async function handleRefresh() {
  await Promise.all([
    channelStore.fetchChannels(),
    channelStore.fetchStats(deviceIdFilter.value || undefined),
  ])
  ElMessage.success('刷新成功')
}

/** 筛选状态变化 */
function handleFilterChange() {
  channelStore.updateFilter({
    deviceId: deviceIdFilter.value || undefined,
    status: statusFilter.value as '0' | '1' | undefined,
  })
}

/** 分页大小变化 */
function handleSizeChange(size: number) {
  pageSize.value = size
  channelStore.updatePagination(1, size)
}

/** 页码变化 */
function handlePageChange(page: number) {
  currentPage.value = page
  channelStore.updatePagination(page, pageSize.value)
}

/** 播放通道 */
function handlePlay(channel: Channel) {
  // 跳转到播放页面，携带设备ID和通道ID参数
  router.push({
    path: '/play',
    query: {
      deviceId: channel.deviceId,
      channelId: channel.channelId,
    },
  })
}

/** 删除通道 */
async function handleDelete(channelId: string, deviceId: string) {
  const success = await channelStore.deleteChannel(channelId, deviceId)
  if (success) {
    ElMessage.success('删除成功')
  }
}

// 监听路由参数变化
watch(
  () => route.query.deviceId,
  (newDeviceId) => {
    if (newDeviceId) {
      deviceIdFilter.value = newDeviceId as string
      channelStore.updateFilter({ deviceId: newDeviceId as string })
    }
  },
  { immediate: true }
)

onMounted(() => {
  channelStore.fetchChannels()
})
</script>