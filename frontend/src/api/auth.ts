import { getData, postData } from './client'

export type CurrentUser = {
  id: string
  username: string
  displayName: string
  roles: string[]
  permissions: string[]
}

export function login(username: string, password: string) {
  return postData<{ token: string; user: CurrentUser }>('/api/auth/login', { username, password })
}

export function logout() {
  return postData<{ success: boolean }>('/api/auth/logout')
}

export function getCurrentUser() {
  return getData<CurrentUser>('/api/auth/me')
}
