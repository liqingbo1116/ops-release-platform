import { baselineMockData } from './mockData/baseline'
import { environmentMockData } from './mockData/environment'
import { getData, getDataWithParams, postData, type PageResult, useMockApi } from './client'

type BaselineItem = typeof baselineMockData.baselineDetail

export type CreateBaselinePayload = {
  sourceEnvironmentId: string
  name: string
  purpose: string
}

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

export async function createBaseline(payload: CreateBaselinePayload) {
  if (!useMockApi) {
    return postData<BaselineItem>('/api/baselines', payload)
  }
  const createdAt = new Date().toISOString()
  const detail = {
    ...baselineMockData.baselineDetail,
    id: `BL-MOCK-${Date.now()}`,
    name: payload.name,
    sourceEnvironmentId: payload.sourceEnvironmentId,
    sourceEnvironmentName: environmentMockData.environments.find((item) => item.id === payload.sourceEnvironmentId)?.name || '未知环境',
    status: 'DRAFT',
    createdBy: 'Mock User',
    createdAt,
    purpose: payload.purpose,
    snapshotSource: `${environmentMockData.environments.find((item) => item.id === payload.sourceEnvironmentId)?.name || '未知环境'}/mock-runtime`,
    snapshotCollectedAt: createdAt,
    snapshotMode: 'MOCK_RUNTIME',
    snapshotTaskId: `snapshot-mock-${Date.now()}`,
  }
  baselineMockData.baselines = [
    {
      id: detail.id,
      name: detail.name,
      sourceEnvironmentId: detail.sourceEnvironmentId,
      sourceEnvironmentName: detail.sourceEnvironmentName,
      serviceCount: detail.items.length,
      createdBy: detail.createdBy,
      createdAt: detail.createdAt,
      status: detail.status,
      purpose: detail.purpose,
      lockedAt: '',
      snapshotSource: detail.snapshotSource,
      snapshotCollectedAt: detail.snapshotCollectedAt,
      snapshotMode: detail.snapshotMode,
    },
    ...baselineMockData.baselines,
  ]
  baselineMockData.baselineDetail = detail
  return detail
}

export async function lockBaseline(id: string) {
  if (!useMockApi) {
    return postData<BaselineItem>(`/api/baselines/${id}/lock`)
  }
  baselineMockData.baselines = baselineMockData.baselines.map((item) =>
    item.id === id
      ? {
          ...item,
          status: 'LOCKED',
          lockedAt: new Date().toISOString(),
        }
      : item,
  )
  if (baselineMockData.baselineDetail.id === id) {
    baselineMockData.baselineDetail = {
      ...baselineMockData.baselineDetail,
      status: 'LOCKED',
      lockedAt: new Date().toISOString(),
    }
  }
  return baselineMockData.baselineDetail
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
