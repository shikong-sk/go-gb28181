<template>
  <div class="tw-p-6">
    <!-- 页面标题 -->
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">设备管理</h1>
      <p class="tw-text-gray-500 tw-mt-1">管理 GB28181 设备注册与状态</p>
    </div>

    <!-- 筛选栏 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4 tw-mb-6">
      <el-form :inline="true" class="tw-flex tw-flex-wrap tw-gap-4">
        <el-form-item label="设备状态">
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
    <div class="tw-grid tw-grid-cols-1 md:tw-grid-cols-3 tw-gap-4 tw-mb-6">
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-blue-100 tw-text-blue-600">
            <el-icon size="24"><Monitor /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">设备总数</p>
            <p class="tw-text-2xl tw-font-bold tw-text-gray-800">{{ deviceStore.stats.total }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-green-100 tw-text-green-600">
            <el-icon size="24"><CircleCheck /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">在线设备</p>
            <p class="tw-text-2xl tw-font-bold tw-text-green-600">{{ deviceStore.stats.online }}</p>
          </div>
        </div>
      </div>
      <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4">
        <div class="tw-flex tw-items-center">
          <div class="tw-p-3 tw-rounded-full tw-bg-red-100 tw-text-red-600">
            <el-icon size="24"><CircleClose /></el-icon>
          </div>
          <div class="tw-ml-4">
            <p class="tw-text-sm tw-text-gray-500">离线设备</p>
            <p class="tw-text-2xl tw-font-bold tw-text-red-600">{{ deviceStore.stats.offline }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 设备列表 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow">
      <el-table
        v-loading="deviceStore.loading"
        :data="deviceStore.devices"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="deviceId" label="设备ID" min-width="180" />
        <el-table-column prop="name" label="设备名称" min-width="150">
          <template #default="{ row }">
            {{ row.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="manufacturer" label="厂商" min-width="120">
          <template #default="{ row }">
            {{ row.manufacturer || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP地址" min-width="140">
          <template #default="{ row }">
            {{ row.ip }}:{{ row.port }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === '1' ? 'success' : 'danger'" size="small">
              {{ row.status === '1' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="channelCount" label="通道数" width="80" align="center" />
        <el-table-column prop="lastKeepaliveTime" label="最后心跳" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.lastKeepaliveTime) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" size="small" link @click="handleViewDetail(row)">
              详情
            </el-button>
            <el-button type="primary" size="small" link @click="handleViewChannels(row)">
              通道
            </el-button>
            <el-button type="success" size="small" link @click="handleSyncCatalog(row)">
              同步
            </el-button>
            <el-popconfirm
              title="确定删除该设备？"
              confirm-button-text="确定"
              cancel-button-text="取消"
              @confirm="handleDelete(row.deviceId)"
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
          :total="deviceStore.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Monitor, CircleCheck, CircleClose, Refresh } from '@element-plus/icons-vue'
import { useDeviceStore } from '@/stores/device'
import type { Device } from '@/types/device'

const router = useRouter()
const deviceStore = useDeviceStore()

const statusFilter = ref<string>('')
const currentPage = ref(1)
const pageSize = ref(20)

/** 格式化时间 */
function formatTime(time: string) {
  if (!time || time === '0001-01-01T00:00:00Z') return '-'
  return new Date(time).toLocaleString('zh-CN')
}

/** 刷新列表 */
async function handleRefresh() {
  await Promise.all([
    deviceStore.fetchDevices(),
    deviceStore.fetchStats(),
  ])
  ElMessage.success('刷新成功')
}

/** 筛选状态变化 */
function handleFilterChange() {
  deviceStore.updateFilter({ status: statusFilter.value as '0' | '1' | undefined })
}

/** 分页大小变化 */
function handleSizeChange(size: number) {
  pageSize.value = size
  deviceStore.updatePagination(1, size)
}

/** 页码变化 */
function handlePageChange(page: number) {
  currentPage.value = page
  deviceStore.updatePagination(page, pageSize.value)
}

/** 查看详情 */
function handleViewDetail(device: Device) {
  router.push(`/devices/${device.deviceId}`)
}

/** 查看通道 */
function handleViewChannels(device: Device) {
  router.push({ path: '/channels', query: { deviceId: device.deviceId } })
}

/** 删除设备 */
async function handleDelete(deviceId: string) {
  const success = await deviceStore.deleteDevice(deviceId)
  if (success) {
    ElMessage.success('删除成功')
  }
}

/** 同步目录 */
async function handleSyncCatalog(device: Device) {
  try {
    const { deviceApi } = await import('@/api/device')
    await deviceApi.syncCatalog(device.deviceId)
    ElMessage.success('目录同步请求已发送')
  } catch {
    ElMessage.error('目录同步失败')
  }
}

onMounted(() => {
  deviceStore.fetchDevices()
  deviceStore.fetchStats()
})
</script>