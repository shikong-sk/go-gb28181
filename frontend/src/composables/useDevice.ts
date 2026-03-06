import { ref } from 'vue'
import { useDeviceStore } from '@/stores/device'
import type { Device, DeviceQuery } from '@/types/device'

export function useDevice() {
  const deviceStore = useDeviceStore()
  const loading = ref(false)
  const deviceDetail = ref<Device | null>(null)

  const fetchDevices = async (params?: DeviceQuery) => {
    loading.value = true
    try {
      return await deviceStore.fetchDevices(params)
    } finally {
      loading.value = false
    }
  }

  const syncCatalog = async (deviceId: string) => {
    const { deviceApi } = await import('@/api/device')
    return deviceApi.syncCatalog(deviceId)
  }

  return {
    deviceStore,
    loading,
    deviceDetail,
    fetchDevices,
    syncCatalog,
  }
}