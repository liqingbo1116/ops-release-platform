import { userMockData } from './mockData/user'
import { getData, type PageResult, useMockApi } from './client'

export async function listUsers() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof userMockData.users[number]>>('/api/users')
    return result.items
  }
  return Promise.resolve(userMockData.users)
}

export async function listRoles() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof userMockData.roles[number]>>('/api/roles')
    return result.items
  }
  return Promise.resolve(userMockData.roles)
}

export async function listPermissions() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof userMockData.permissions[number]>>('/api/permissions')
    return result.items
  }
  return Promise.resolve(userMockData.permissions)
}
