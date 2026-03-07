<template>
  <div class="tw-p-6">
    <!-- 页面标题 -->
    <div class="tw-mb-6">
      <h1 class="tw-text-2xl tw-font-bold tw-text-gray-800">报警记录</h1>
      <p class="tw-text-gray-500 tw-mt-1">查看设备报警信息</p>
    </div>

    <!-- 配置卡片 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4 tw-mb-6">
      <div class="tw-flex tw-items-center tw-justify-between">
        <div class="tw-flex tw-items-center tw-gap-4">
          <span class="tw-text-gray-600">记录保存:</span>
          <el-tag :type="config.enabled ? 'success' : 'info'">
            {{ config.enabled ? '已启用' : '已禁用' }}
          </el-tag>
          <span class="tw-text-gray-600" v-if="config.enabled">
            保留最近 {{ config.retention_days }} 天
          </span>
        </div>
        <el-button type="danger" @click="handleDeleteAll" :disabled="loading">
          <el-icon class="tw-mr-1"><Delete /></el-icon>
          清空记录
        </el-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow tw-p-4 tw-mb-6">
      <el-form :inline="true" class="tw-flex tw-flex-wrap tw-gap-4">
        <el-form-item label="设备ID">
          <el-input
            v-model="query.device_id"
            placeholder="请输入设备ID"
            clearable
            style="width: 200px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="报警级别">
          <el-select
            v-model="query.priority"
            placeholder="全部"
            clearable
            style="width: 120px"
          >
            <el-option label="一级" value="1" />
            <el-option label="二级" value="2" />
            <el-option label="三级" value="3" />
          </el-select>
        </el-form-item>
        <el-form-item label="开始时间">
          <el-date-picker
            v-model="query.start_time"
            type="datetime"
            placeholder="选择开始时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker
            v-model="query.end_time"
            type="datetime"
            placeholder="选择结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            style="width: 180px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleSearch">
            <el-icon class="tw-mr-1"><Search /></el-icon>
            查询
          </el-button>
          <el-button @click="handleReset">
            <el-icon class="tw-mr-1"><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 报警列表 -->
    <div class="tw-bg-white tw-rounded-lg tw-shadow">
      <el-table v-loading="loading" :data="alarms" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column prop="deviceId" label="设备ID" min-width="180" />
        <el-table-column prop="alarmPriority" label="级别" width="80" align="center">
          <template #default="{ row }">
            <el-tag :type="getPriorityType(row.alarmPriority)" size="small">
              {{ getPriorityLabel(row.alarmPriority) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="alarmMethod" label="报警方式" width="100" align="center">
          <template #default="{ row }">
            {{ getMethodLabel(row.alarmMethod) }}
          </template>
        </el-table-column>
        <el-table-column prop="alarmTime" label="报警时间" min-width="180" />
        <el-table-column prop="alarmDescription" label="描述" min-width="200">
          <template #default="{ row }">
            {{ row.alarmDescription || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="接收时间" min-width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-popconfirm
              title="确定删除该报警记录？"
              confirm-button-text="确定"
              cancel-button-text="取消"
              @confirm="handleDelete(row.id)"
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
          v-model:current-page="query.page"
          v-model:page-size="query.page_size"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchAlarms"
          @current-change="fetchAlarms"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Search, Refresh } from '@element-plus/icons-vue'
import { alarmApi, type Alarm, type AlarmConfig, type AlarmQuery } from '@/api/alarm'

const loading = ref(false)
const alarms = ref<Alarm[]>([])
const total = ref(0)

const query = reactive<AlarmQuery>({
  page: 1,
  page_size: 20,
  device_id: '',
  priority: '',
  start_time: '',
  end_time: '',
})

const config = reactive<AlarmConfig>({
  enabled: true,
  retention_days: 3,
})

/** 获取报警级别标签类型 */
function getPriorityType(priority: string): 'danger' | 'warning' | 'info' {
  const map: Record<string, 'danger' | 'warning' | 'info'> = {
    '1': 'danger',
    '2': 'warning',
    '3': 'info',
  }
  return map[priority] || 'info'
}

/** 获取报警级别文本 */
function getPriorityLabel(priority: string): string {
  const map: Record<string, string> = {
    '1': '一级',
    '2': '二级',
    '3': '三级',
  }
  return map[priority] || priority
}

/** 获取报警方式文本 */
function getMethodLabel(method: string): string {
  const map: Record<string, string> = {
    '1': '电话报警',
    '2': '设备报警',
    '3': '短信报警',
    '4': 'GPS报警',
    '5': '视频报警',
    '6': '设备故障',
    '7': '其他报警',
  }
  return map[method] || method
}

/** 获取报警配置 */
async function fetchConfig() {
  try {
    const res = await alarmApi.getConfig()
    if (res.data) {
      config.enabled = res.data.enabled
      config.retention_days = res.data.retention_days
    }
  } catch {
    // 忽略错误
  }
}

/** 获取报警列表 */
async function fetchAlarms() {
  loading.value = true
  try {
    const params: AlarmQuery = { ...query }
    // 移除空值
    Object.keys(params).forEach((key) => {
      const k = key as keyof AlarmQuery
      if (!params[k]) delete params[k]
    })
    const res = await alarmApi.getList(params)
    if (res.data) {
      alarms.value = res.data.list || []
      total.value = res.data.total || 0
    }
  } catch {
    ElMessage.error('获取报警列表失败')
  } finally {
    loading.value = false
  }
}

/** 搜索 */
function handleSearch() {
  query.page = 1
  fetchAlarms()
}

/** 重置 */
function handleReset() {
  query.page = 1
  query.page_size = 20
  query.device_id = ''
  query.priority = ''
  query.start_time = ''
  query.end_time = ''
  fetchAlarms()
}

/** 删除单条 */
async function handleDelete(id: number) {
  try {
    await alarmApi.delete(id)
    ElMessage.success('删除成功')
    fetchAlarms()
  } catch {
    ElMessage.error('删除失败')
  }
}

/** 清空所有 */
async function handleDeleteAll() {
  try {
    await ElMessageBox.confirm('确定要清空所有报警记录吗？此操作不可恢复！', '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await alarmApi.deleteAll()
    ElMessage.success('已清空所有报警记录')
    fetchAlarms()
  } catch {
    // 用户取消或失败
  }
}

onMounted(() => {
  fetchConfig()
  fetchAlarms()
})
</script>