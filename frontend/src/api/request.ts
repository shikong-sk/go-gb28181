import axios, { type AxiosInstance, type AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'

/** API 响应结构 */
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
}

/** 创建 Axios 实例 */
const instance: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

/** 请求拦截器 */
instance.interceptors.request.use(
  (config) => {
    // 可以在这里添加 token 等认证信息
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

/** 响应拦截器 */
instance.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data

    // 业务错误处理
    if (res.code !== 200) {
      ElMessage.error(res.message || '请求失败')
      return Promise.reject(new Error(res.message || '请求失败'))
    }

    return response
  },
  (error) => {
    // HTTP 错误处理
    const message = error.response?.data?.message || error.message || '网络错误'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

/** GET 请求 */
export function get<T>(url: string, params?: Record<string, unknown>): Promise<ApiResponse<T>> {
  return instance.get(url, { params }).then((res) => res.data)
}

/** POST 请求 */
export function post<T>(url: string, data?: unknown): Promise<ApiResponse<T>> {
  return instance.post(url, data).then((res) => res.data)
}

/** DELETE 请求 */
export function del<T>(url: string, params?: Record<string, unknown>): Promise<ApiResponse<T>> {
  return instance.delete(url, { params }).then((res) => res.data)
}

/** PUT 请求 */
export function put<T>(url: string, data?: unknown): Promise<ApiResponse<T>> {
  return instance.put(url, data).then((res) => res.data)
}

export default instance
