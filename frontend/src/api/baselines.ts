import { baselineMockData } from './mockData/baseline'
import { environmentMockData } from './mockData/environment'
import { getData, getDataWithParams, postData, type PageResult, useMockApi } from './client'

export async function listBaselines() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof baselineMockData.baselines[number]>>('/api/baselines')
    return result.items
  }
  return Promise.resolve(baselineMockData.baselines)
}

export function getBaselineDetail(id = 'BL-20260607-0001') {
  if (!useMockApi) {
    return getData<typeof baselineMockData.baselineDetail>(`/api/baselines/${id}`)
  }
  return Promise.resolve(baselineMockData.baselineDetail)
}

export function getBaselineCompare(id = 'BL-20260607-0001', targetEnvironmentId?: string) {
  if (!useMockApi) {
    return postData<typeof baselineMockData.diffResult>(`/api/baselines/${id}/compare`, targetEnvironmentId ? { targetEnvironmentId } : {})
  }
  return Promise.resolve({
    ...baselineMockData.diffResult,
    targetEnvironmentId: targetEnvironmentId || baselineMockData.diffResult.targetEnvironmentId,
  })
}

export function listBaselineTargetEnvironments() {
  if (!useMockApi) {
    return getDataWithParams<PageResult<typeof environmentMockData.environments[number]>>('/api/environments')
      .then((result) => result.items)
  }
  return Promise.resolve(environmentMockData.environments)
}
