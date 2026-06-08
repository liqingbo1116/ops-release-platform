import { defineStore } from 'pinia'
import { releaseMockData } from '@/api/mockData/release'

export const useReleaseStore = defineStore('release', {
  state: () => ({
    detail: releaseMockData.releaseDetail,
  }),
})
