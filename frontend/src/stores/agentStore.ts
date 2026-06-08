import { defineStore } from 'pinia'
import { agentMockData } from '@/api/mockData/agent'

export const useAgentStore = defineStore('agent', {
  state: () => ({
    items: agentMockData.agents,
  }),
})
