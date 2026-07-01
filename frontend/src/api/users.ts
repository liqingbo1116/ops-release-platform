import { getData, type PageResult } from './client'

export type UserInfo = {
  id: string
  username: string
  displayName: string
  roles: string[]
  status: string
  lastLoginAt?: string
}

export type RoleInfo = {
  id: string
  name: string
  description?: string
  permissions?: string[]
}

export type EnvironmentPermission = {
  id: string
  roleId: string
  roleName: string
  environmentId: string
  environmentName: string
  actions: string[]
}

export async function listUsers() {
  const result = await getData<PageResult<UserInfo>>('/api/users')
  return result.items
}

export async function listRoles() {
  const result = await getData<PageResult<RoleInfo>>('/api/roles')
  return result.items
}

export async function listPermissions() {
  const result = await getData<PageResult<EnvironmentPermission>>('/api/permissions')
  return result.items
}
