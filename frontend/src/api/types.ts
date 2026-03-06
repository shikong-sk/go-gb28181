// API类型定义
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

export interface PageResponse<T> {
  list: T[]
  total: number
}