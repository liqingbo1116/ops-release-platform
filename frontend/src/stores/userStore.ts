import { defineStore } from 'pinia'
import type { EnvironmentPermission, RoleInfo, UserInfo } from '@/api/users'

export const useUserStore = defineStore('user', {
  state: () => ({
    users: [] as UserInfo[],
    roles: [] as RoleInfo[],
    permissions: [] as EnvironmentPermission[],
  }),
})
