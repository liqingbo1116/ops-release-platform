import { mockData } from './mockData'
import { getData, type PageResult, useMockApi } from './client'

export async function listAgents() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.agents[number]>>('/api/agents')
    return result.items
  }
  return Promise.resolve(mockData.agents)
}
