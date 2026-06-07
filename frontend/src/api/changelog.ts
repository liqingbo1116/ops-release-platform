import { mockData } from './mockData'
import { getData, type PageResult, useMockApi } from './client'

export async function listChangelog() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.changelog[number]>>('/api/changelog')
    return result.items
  }
  return Promise.resolve(mockData.changelog)
}
