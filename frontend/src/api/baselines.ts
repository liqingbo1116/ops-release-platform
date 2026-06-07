import { mockData } from './mockData'
import { getData, postData, type PageResult, useMockApi } from './client'

export async function listBaselines() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.baselines[number]>>('/api/baselines')
    return result.items
  }
  return Promise.resolve(mockData.baselines)
}

export function getBaselineDetail() {
  if (!useMockApi) {
    return getData<typeof mockData.baselineDetail>('/api/baselines/BL-20260607-0001')
  }
  return Promise.resolve(mockData.baselineDetail)
}

export function getBaselineCompare() {
  if (!useMockApi) {
    return postData<typeof mockData.diffResult>('/api/baselines/BL-20260607-0001/compare')
  }
  return Promise.resolve(mockData.diffResult)
}
