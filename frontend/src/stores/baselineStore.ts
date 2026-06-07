import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useBaselineStore = defineStore('baseline', {
  state: () => ({
    items: mockData.baselines,
    detail: mockData.baselineDetail,
    diff: mockData.diffResult,
  }),
})
