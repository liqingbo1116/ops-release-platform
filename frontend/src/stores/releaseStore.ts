import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useReleaseStore = defineStore('release', {
  state: () => ({
    detail: mockData.releaseDetail,
  }),
})
