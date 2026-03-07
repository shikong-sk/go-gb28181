<template>
  <div class="tw-h-full tw-flex tw-flex-col">
    <div class="tw-h-16 tw-flex tw-items-center tw-justify-center tw-text-white tw-text-xl tw-font-bold">
      GB28181
    </div>
    <el-menu
      :default-active="$route.path"
      class="tw-flex-1 tw-border-none"
      background-color="#1f2937"
      text-color="#fff"
      active-text-color="#409EFF"
      router
    >
      <el-menu-item index="/">
        <el-icon><HomeFilled /></el-icon>
        <span>首页</span>
      </el-menu-item>
      <el-menu-item index="/devices">
        <el-icon><VideoCamera /></el-icon>
        <span>设备管理</span>
      </el-menu-item>
      <el-menu-item index="/channels">
        <el-icon><List /></el-icon>
        <span>通道管理</span>
      </el-menu-item>
      <el-menu-item index="/play">
        <el-icon><VideoPlay /></el-icon>
        <span>视频播放</span>
      </el-menu-item>
      <el-menu-item index="/alarms">
        <el-icon><Bell /></el-icon>
        <span>报警记录</span>
      </el-menu-item>
    </el-menu>
  </div>
</template>

<script setup lang="ts">
import { HomeFilled, VideoCamera, List, VideoPlay, Bell } from '@element-plus/icons-vue'
</script>