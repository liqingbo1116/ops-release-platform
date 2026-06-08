import { authMockData } from './mockData/auth'
import { getData, postData, useMockApi } from './client'

export function login(username: string, password: string) {
  if (!useMockApi) {
    return postData<{ token: string; user: typeof authMockData.currentUser }>('/api/auth/login', { username, password })
  }
  return Promise.resolve({
    token: 'mock-token-admin',
    user: authMockData.currentUser,
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
    return getData<typeof authMockData.currentUser>('/api/auth/me')
  }
  return Promise.resolve(authMockData.currentUser)
}
