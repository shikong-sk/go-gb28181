import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import path from 'path'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      imports: ['vue', 'vue-router', 'pinia'],
      resolvers: [ElementPlusResolver()],
      dts: 'src/types/auto-imports.d.ts',
    }),
    Components({
      resolvers: [ElementPlusResolver()],
      dts: 'src/types/components.d.ts',
    }),
  ],
    // 配置esbuild支持装饰器
  esbuild: {
    tsconfigRaw: {
      compilerOptions: {
        experimentalDecorators: true,
        useDefineForClassFields: false,
      },
    },
  },
  // 优化jessibuca相关包的依赖
  optimizeDeps: {
    include: [
      'reflect-metadata',
      'afsm',
      'eventemitter3',
      'oput',
      'jv4-connection',
      'jv4-demuxer',
      'jv4-decoder',
    ],
    // 强制预构建这些包
    force: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      // 使用预编译版本
      'jv4-connection': path.resolve(__dirname, 'node_modules/jv4-connection/dist/index.js'),
      'jv4-decoder': path.resolve(__dirname, 'node_modules/jv4-decoder/dist/index.js'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://10.10.10.30:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://10.10.10.30:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
})