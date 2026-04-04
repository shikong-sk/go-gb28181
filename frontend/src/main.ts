import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import './assets/styles/tailwind.css'
import './assets/styles/index.css'
import App from './App.vue'
import router from './router'

// 导入 flv.js 用于视频播放
import flvjs from 'flv.js'
// @ts-expect-error flv.js 挂载到 window
window.flvjs = flvjs

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(ElementPlus, { locale: zhCn })

app.mount('#app')