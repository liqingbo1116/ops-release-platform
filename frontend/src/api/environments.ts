import { environmentMockData } from './mockData/environment'
import { getData, type PageResult, useMockApi } from './client'

export async function listEnvironments() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof environmentMockData.environments[number]>>('/api/environments')
    return result.items
  }
  return Promise.resolve(environmentMockData.environments)
}
