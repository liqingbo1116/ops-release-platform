import { agentMockData } from './mockData/agent'
import { getData, type PageResult, useMockApi } from './client'

export type AgentInfo = (typeof agentMockData.agents)[number]

export async function listAgents(): Promise<AgentInfo[]> {
  if (!useMockApi) {
    const result = await getData<PageResult<AgentInfo>>('/api/agents')
    return result.items
  }
  return Promise.resolve(agentMockData.agents)
}
