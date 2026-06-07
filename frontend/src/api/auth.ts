import { mockData } from './mockData'
import { getData, postData, useMockApi } from './client'

export function login(username: string, password: string) {
  if (!useMockApi) {
    return postData<{ token: string; user: typeof mockData.currentUser }>('/api/auth/login', { username, password })
  }
  return Promise.resolve({
    token: 'mock-token-admin',
    user: mockData.currentUser,
  })
}

export function logout() {
  if (!useMockApi) {
    return postData<{ success: boolean }>('/api/auth/logout')
  }
  return Promise.resolve({ success: true })
}

export function getCurrentUser() {
  if (!useMockApi) {
    return getData<typeof mockData.currentUser>('/api/auth/me')
  }
  return Promise.resolve(mockData.currentUser)
}
