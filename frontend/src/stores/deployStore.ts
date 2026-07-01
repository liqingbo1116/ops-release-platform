import { defineStore } from 'pinia'
import type { DeployDetail, DeployTask } from '@/api/deployTasks'

export const useDeployStore = defineStore('deploy', {
  state: () => ({
    items: [] as DeployTask[],
    detail: null as DeployDetail | null,
  }),
})
