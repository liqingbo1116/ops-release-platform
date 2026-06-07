import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useChangelogStore = defineStore('changelog', {
  state: () => ({
    items: mockData.changelog,
  }),
})
