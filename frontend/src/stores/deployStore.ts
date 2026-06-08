import { defineStore } from 'pinia'
import { deployMockData } from '@/api/mockData/deploy'

export const useDeployStore = defineStore('deploy', {
  state: () => ({
    items: deployMockData.deployTasks,
    detail: deployMockData.deployDetail,
  }),
})
