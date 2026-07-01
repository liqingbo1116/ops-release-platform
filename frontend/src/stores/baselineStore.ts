import { defineStore } from 'pinia'
import type { BaselineDetailItem, BaselineDiffResult, BaselineListItem } from '@/api/baselines'

export const useBaselineStore = defineStore('baseline', {
  state: () => ({
    items: [] as BaselineListItem[],
    detail: null as BaselineDetailItem | null,
    diff: null as BaselineDiffResult | null,
  }),
})
