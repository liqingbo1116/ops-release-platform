import { changelogMockData } from './mockData/changelog'
import { getData, type PageResult, useMockApi } from './client'

export async function listChangelog() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof changelogMockData.changelog[number]>>('/api/changelog')
    return result.items
  }
  return Promise.resolve(changelogMockData.changelog)
}
