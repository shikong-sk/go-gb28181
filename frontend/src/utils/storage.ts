// 本地存储工具函数
const STORAGE_PREFIX = 'gb28181_'

export const storage = {
  get<T>(key: string): T | null {
    const value = localStorage.getItem(STORAGE_PREFIX + key)
    if (value) {
      try {
        return JSON.parse(value) as T
      } catch {
        return value as unknown as T
      }
    }
    return null
  },

  set<T>(key: string, value: T): void {
    localStorage.setItem(STORAGE_PREFIX + key, JSON.stringify(value))
  },

  remove(key: string): void {
    localStorage.removeItem(STORAGE_PREFIX + key)
  },

  clear(): void {
    Object.keys(localStorage)
      .filter(key => key.startsWith(STORAGE_PREFIX))
      .forEach(key => localStorage.removeItem(key))
  },
}