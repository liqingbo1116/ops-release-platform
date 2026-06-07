import { mockData } from './mockData'
import { getData, type PageResult, useMockApi } from './client'

export async function listEnvironments() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.environments[number]>>('/api/environments')
    return result.items
  }
  return Promise.resolve(mockData.environments)
}
