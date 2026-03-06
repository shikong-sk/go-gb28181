<template>
  <el-container class="tw-h-full">
    <el-aside width="200px" class="tw-bg-gray-800">
      <AppSidebar />
    </el-aside>
    <el-container>
      <el-header class="tw-bg-white tw-border-b">
        <AppHeader />
      </el-header>
      <el-main class="tw-bg-gray-100">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import AppHeader from './AppHeader.vue'
import AppSidebar from './AppSidebar.vue'
</script>