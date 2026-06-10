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

export class ApiClientError extends Error {
  status?: number
  code?: string
  requestId?: string

  constructor(message: string, options?: { status?: number; code?: string; requestId?: string }) {
    super(message)
    this.name = 'ApiClientError'
    this.status = options?.status
    this.code = options?.code
    this.requestId = options?.requestId
  }
}

function normalizeError(error: unknown): ApiClientError {
  if (error instanceof ApiClientError) {
    return error
  }

  if (axios.isAxiosError<ApiResponse<unknown>>(error)) {
    const payload = error.response?.data
    return new ApiClientError(payload?.message || error.message || 'Request failed', {
      status: error.response?.status,
      code: payload?.code,
      requestId: payload?.requestId,
    })
  }

  if (error instanceof Error) {
    return new ApiClientError(error.message)
  }

  return new ApiClientError('Request failed')
}

export async function getData<T>(url: string): Promise<T> {
  try {
    const response = await apiClient.get<ApiResponse<T>>(url)
    return response.data.data
  } catch (error) {
    throw normalizeError(error)
  }
}

export async function postData<T>(url: string, body?: unknown): Promise<T> {
  try {
    const response = await apiClient.post<ApiResponse<T>>(url, body ?? {})
    return response.data.data
  } catch (error) {
    throw normalizeError(error)
  }
}

export async function putData<T>(url: string, body?: unknown): Promise<T> {
  try {
    const response = await apiClient.put<ApiResponse<T>>(url, body ?? {})
    return response.data.data
  } catch (error) {
    throw normalizeError(error)
  }
}

export async function getDataWithParams<T>(url: string, params?: Record<string, unknown>): Promise<T> {
  try {
    const response = await apiClient.get<ApiResponse<T>>(url, { params })
    return response.data.data
  } catch (error) {
    throw normalizeError(error)
  }
}
