/**
 * Playwright 调试脚本 - 测试前端功能
 * 使用 chrome-devtools 协议调试
 */
import { chromium } from 'playwright'

async function debug() {
  console.log('启动浏览器调试...')

  const browser = await chromium.launch({
    headless: false,
    devtools: true,
    args: ['--auto-open-devtools-for-tabs']
  })

  const page = await browser.newPage()

  // 监听控制台消息
  page.on('console', msg => {
    const type = msg.type()
    const text = msg.text()
    if (type === 'error') {
      console.log(`[ERROR] ${text}`)
    } else if (type === 'warning') {
      console.log(`[WARNING] ${text}`)
    } else {
      console.log(`[LOG] ${text}`)
    }
  })

  // 监听网络请求
  page.on('request', req => {
    if (req.url().includes('/api/')) {
      console.log(`[REQUEST] ${req.method()} ${req.url()}`)
    }
  })

  page.on('response', async res => {
    if (res.url().includes('/api/')) {
      console.log(`[RESPONSE] ${res.status()} ${res.url()}`)
      try {
        const body = await res.json()
        console.log(`[RESPONSE BODY]`, JSON.stringify(body, null, 2).slice(0, 500))
      } catch (e) {
        // 忽略非JSON响应
      }
    }
  })

  // 访问前端页面
  const baseUrl = 'http://localhost:5177'
  console.log(`访问页面: ${baseUrl}`)
  await page.goto(baseUrl)

  // 等待页面加载
  await page.waitForLoadState('networkidle')

  // 获取页面状态
  const title = await page.title()
  console.log(`页面标题: ${title}`)

  // 截图
  await page.screenshot({ path: 'debug-screenshot.png', fullPage: true })
  console.log('截图已保存: debug-screenshot.png')

  // 检查关键元素
  const navItems = await page.$$eval('nav a', els => els.map(el => el.textContent))
  console.log('导航项:', navItems)

  // 保持浏览器打开
  console.log('\n浏览器保持打开，按Ctrl+C退出...')
  await page.waitForTimeout(600000) // 10分钟

  await browser.close()
}

debug().catch(console.error)