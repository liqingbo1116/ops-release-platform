import { environmentMockData } from './mockData/environment'
import { getData, type PageResult, useMockApi } from './client'

export type EnvironmentInfo = (typeof environmentMockData.environments)[number]

export async function listEnvironments(): Promise<EnvironmentInfo[]> {
  if (!useMockApi) {
    const result = await getData<PageResult<EnvironmentInfo>>('/api/environments')
    return result.items
  }
  return Promise.resolve(environmentMockData.environments)
}
