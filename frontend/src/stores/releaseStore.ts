import { defineStore } from 'pinia'
import type { ReleaseDetail } from '@/api/releases'

export const useReleaseStore = defineStore('release', {
  state: () => ({
    detail: null as ReleaseDetail | null,
  }),
})
