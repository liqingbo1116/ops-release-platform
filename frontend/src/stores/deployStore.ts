import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useDeployStore = defineStore('deploy', {
  state: () => ({
    items: mockData.deployTasks,
    detail: mockData.deployDetail,
  }),
})
