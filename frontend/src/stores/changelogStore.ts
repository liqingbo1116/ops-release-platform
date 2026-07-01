import { defineStore } from 'pinia'
import type { ChangelogEntry } from '@/api/changelog'

export const useChangelogStore = defineStore('changelog', {
  state: () => ({
    items: [] as ChangelogEntry[],
  }),
})
