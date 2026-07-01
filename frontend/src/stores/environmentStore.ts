import { defineStore } from 'pinia'
import type { EnvironmentInfo } from '@/api/environments'

export const useEnvironmentStore = defineStore('environment', {
  state: () => ({
    items: [] as EnvironmentInfo[],
  }),
})
