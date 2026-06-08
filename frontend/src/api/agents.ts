import { agentMockData } from './mockData/agent'
import { getData, type PageResult, useMockApi } from './client'

export async function listAgents() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof agentMockData.agents[number]>>('/api/agents')
    return result.items
  }
  return Promise.resolve(agentMockData.agents)
}
