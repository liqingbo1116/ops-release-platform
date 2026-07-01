import { getData, getDataWithParams, postData, type PageResult } from './client'
import type { EnvironmentInfo } from './environments'

export type BaselineListItem = {
  id: string
  name: string
  sourceEnvironmentId?: string
  sourceEnvironmentName: string
  serviceCount: number
  createdBy: string
  createdAt: string
  status: string
  purpose: string
  lockedAt?: string
  snapshotSource?: string
  snapshotCollectedAt?: string
  snapshotMode?: string
}

export type BaselineDetailItem = BaselineListItem & {
  snapshotTaskId?: string
  items: Array<{
    serviceId: string
    serviceName: string
    namespace: string
    workloadName: string
    workloadType: string
    tag: string
    digest: string
    replicas: number
    readyReplicas: number
    healthStatus: string
  }>
}

export type BaselineDiffResult = {
  baselineId: string
  sourceBaselineId?: string
  targetEnvironmentId: string
  summary: Record<string, number>
  items: BaselineDiffItem[]
}

export type BaselineDiffItem = {
  serviceId: string
  serviceName: string
  namespace: string
  sourceTag?: string
  targetTag?: string
  diffStatus: string
  publishable: boolean
  [key: string]: unknown
}

export type CreateBaselinePayload = {
  sourceEnvironmentId: string
  name: string
  purpose: string
}

export async function listBaselines() {
  const result = await getData<PageResult<BaselineListItem>>('/api/baselines')
  return result.items
}

export function getBaselineDetail(id: string) {
  return getData<BaselineDetailItem>(`/api/baselines/${id}`)
}

export async function createBaseline(payload: CreateBaselinePayload) {
  return postData<BaselineDetailItem>('/api/baselines', payload)
}

export async function lockBaseline(id: string) {
  return postData<BaselineDetailItem>(`/api/baselines/${id}/lock`)
}

export function getBaselineCompare(id: string, targetEnvironmentId?: string) {
  return postData<BaselineDiffResult>(`/api/baselines/${id}/compare`, targetEnvironmentId ? { targetEnvironmentId } : {})
}

export function listBaselineTargetEnvironments() {
  return getDataWithParams<PageResult<EnvironmentInfo>>('/api/environments').then((result) => result.items)
}
