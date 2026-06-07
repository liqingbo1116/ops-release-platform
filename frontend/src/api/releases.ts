import { mockData } from './mockData'
import { getData, useMockApi } from './client'

export function getReleaseDetail() {
  if (!useMockApi) {
    return getData<typeof mockData.releaseDetail>('/api/releases/REL-20260607-031')
  }
  return Promise.resolve(mockData.releaseDetail)
}
