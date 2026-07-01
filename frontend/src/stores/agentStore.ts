import { defineStore } from 'pinia'
import type { AgentInfo } from '@/api/agents'

export const useAgentStore = defineStore('agent', {
  state: () => ({
    items: [] as AgentInfo[],
  }),
})
