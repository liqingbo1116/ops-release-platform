import { mockData } from './mockData'
import { getData, type PageResult, useMockApi } from './client'

export async function listUsers() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.users[number]>>('/api/users')
    return result.items
  }
  return Promise.resolve(mockData.users)
}

export async function listRoles() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.roles[number]>>('/api/roles')
    return result.items
  }
  return Promise.resolve(mockData.roles)
}

export async function listPermissions() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.permissions[number]>>('/api/permissions')
    return result.items
  }
  return Promise.resolve(mockData.permissions)
}
