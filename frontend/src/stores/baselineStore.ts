import { defineStore } from 'pinia'
import { baselineMockData } from '@/api/mockData/baseline'

export const useBaselineStore = defineStore('baseline', {
  state: () => ({
    items: baselineMockData.baselines,
    detail: baselineMockData.baselineDetail,
    diff: baselineMockData.diffResult,
  }),
})
