import { mockData } from './mockData'

export function listAgents() {
  return Promise.resolve(mockData.agents)
}
