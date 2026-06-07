import { defineStore } from 'pinia'
import { mockData } from '@/api/mockData'

export const useAgentStore = defineStore('agent', {
  state: () => ({
    items: mockData.agents,
  }),
})
