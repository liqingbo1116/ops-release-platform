import { defineStore } from 'pinia'
import { environmentMockData } from '@/api/mockData/environment'

export const useEnvironmentStore = defineStore('environment', {
  state: () => ({
    items: environmentMockData.environments,
  }),
})
