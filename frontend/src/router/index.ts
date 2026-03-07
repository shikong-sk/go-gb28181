import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Dashboard',
    component: () => import('@/views/dashboard/index.vue'),
    meta: { title: '仪表盘' },
  },
  {
    path: '/devices',
    name: 'Devices',
    component: () => import('@/views/devices/index.vue'),
    meta: { title: '设备管理' },
  },
  {
    path: '/devices/:deviceId',
    name: 'DeviceDetail',
    component: () => import('@/views/devices/detail.vue'),
    meta: { title: '设备详情' },
  },
  {
    path: '/channels',
    name: 'Channels',
    component: () => import('@/views/channels/index.vue'),
    meta: { title: '通道管理' },
  },
  {
    path: '/play',
    name: 'Play',
    component: () => import('@/views/play/index.vue'),
    meta: { title: '视频播放' },
  },
  {
    path: '/alarms',
    name: 'Alarms',
    component: () => import('@/views/alarms/index.vue'),
    meta: { title: '报警记录' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router