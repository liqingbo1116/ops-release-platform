import { defineStore } from 'pinia'
import { userMockData } from '@/api/mockData/user'

export const useUserStore = defineStore('user', {
  state: () => ({
    users: userMockData.users,
    roles: userMockData.roles,
    permissions: userMockData.permissions,
  }),
})
