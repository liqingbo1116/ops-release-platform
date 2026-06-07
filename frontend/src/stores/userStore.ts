import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useUserStore = defineStore('user', {
  state: () => ({
    users: mockData.users,
    roles: mockData.roles,
    permissions: mockData.permissions,
  }),
})
