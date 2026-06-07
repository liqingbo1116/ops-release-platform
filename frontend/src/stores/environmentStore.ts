import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useEnvironmentStore = defineStore('environment', {
  state: () => ({
    items: mockData.environments,
  }),
})
