<template>
  <AppLayout />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useDeviceStore } from '@/stores/device'

const deviceStore = useDeviceStore()

// 应用启动时连接 WebSocket
onMounted(() => {
  deviceStore.connectWebSocket()
})

// 应用卸载时断开 WebSocket
onUnmounted(() => {
  deviceStore.disconnectWebSocket()
})
</script>

<style>
#app {
  height: 100vh;
}
</style>