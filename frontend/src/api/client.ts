import axios from 'axios'

export const useMockApi = import.meta.env.VITE_USE_MOCK === 'true'

export const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:8080',
  timeout: 10000,
})

export type ApiResponse<T> = {
  code: string
  message: string
  data: T
  requestId: string
}

export type PageResult<T> = {
  items: T[]
  page: number
  pageSize: number
  total: number
}

export async function getData<T>(url: string): Promise<T> {
  const response = await apiClient.get<ApiResponse<T>>(url)
  return response.data.data
}

export async function postData<T>(url: string, body?: unknown): Promise<T> {
  const response = await apiClient.post<ApiResponse<T>>(url, body ?? {})
  return response.data.data
}
