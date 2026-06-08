import { defineStore } from 'pinia'
import { changelogMockData } from '@/api/mockData/changelog'

export const useChangelogStore = defineStore('changelog', {
  state: () => ({
    items: changelogMockData.changelog,
  }),
})
