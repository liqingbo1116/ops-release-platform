import { releaseMockData } from './mockData/release'
import { getData, getDataWithParams, postData, type PageResult, useMockApi } from './client'

export type CreateReleaseRequest = {
  type: string
  sourceBaselineId?: string
  targetEnvironmentId: string
  agentId: string
  serviceIds: string[]
  releaseSource?: 'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE'
  image?: {
    repository: string
    tag: string
    digest?: string
  }
  jenkins?: {
    jobName: string
    branch: string
    parameters?: Record<string, string>
  }
  options: Record<string, boolean>
}

export type CreateReleaseResult = {
  id: string
  status: string
  executionMode?: string
  agentTaskId?: string
  releaseSource?: 'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE'
  buildId?: string
  buildStatus?: string
  buildUrl?: string
  createdAt: string
}

export type ReleaseImageTag = {
  tag: string
  digest?: string
  updatedAt?: string
}

export type ReleaseSourceService = {
  serviceId: string
  serviceName: string
  namespace: string
  workloadName: string
  workloadType: string
  imageRepository: string
  tags: ReleaseImageTag[]
  publishable: boolean
  message?: string
}

export type ReleaseSource = {
  environmentId: string
  services: ReleaseSourceService[]
  jenkinsJobs: string[]
}

export type ReleaseActionResult = {
  releaseId: string
  action: string
  status: string
  message?: string
  updatedAt?: string
}

export function listReleaseSources(environmentId: string, keyword = '') {
  return getDataWithParams<ReleaseSource>('/api/release-sources', { environmentId, keyword })
}

export function listReleases() {
  if (!useMockApi) {
    return getData<PageResult<typeof releaseMockData.releases[number]>>('/api/releases').then((result) => result.items)
  }
  return Promise.resolve(releaseMockData.releases)
}

export function getReleaseDetail(id = 'REL-20260607-031') {
  if (!useMockApi) {
    return getData<typeof releaseMockData.releaseDetail>(`/api/releases/${id}`)
  }
  return Promise.resolve(releaseMockData.releaseDetail)
}

export function createRelease(body: CreateReleaseRequest) {
  if (!useMockApi) {
    return postData<CreateReleaseResult>('/api/releases', body)
  }
  return Promise.resolve<CreateReleaseResult>({
    id: 'REL-20260607-MOCK',
    status: body.releaseSource === 'LOCAL_HARBOR_IMAGE' ? 'PENDING_IMAGE_SYNC' : 'JENKINS_QUEUED',
    executionMode: body.releaseSource === 'LOCAL_HARBOR_IMAGE' ? 'AGENT_IMAGE_SYNC' : 'JENKINS_AGENT',
    agentTaskId: 'REL-20260607-MOCK',
    releaseSource: body.releaseSource,
    buildId: body.releaseSource === 'LOCAL_HARBOR_IMAGE' ? 'SYNC-MOCK-20260607' : 'BUILD-MOCK-20260607',
    buildStatus: body.releaseSource === 'LOCAL_HARBOR_IMAGE' ? 'SUCCESS' : 'QUEUED',
    buildUrl: body.releaseSource === 'JENKINS_JOB' ? 'https://jenkins.local/job/mock-service-release/1' : undefined,
    createdAt: new Date().toISOString(),
  })
}

export function retryRelease(id: string) {
  if (!useMockApi) {
    return postData<ReleaseActionResult>(`/api/releases/${id}/retry`)
  }
  return Promise.resolve<ReleaseActionResult>({
    releaseId: id,
    action: 'retry',
    status: 'RUNNING',
    message: '已提交失败重试',
    updatedAt: new Date().toISOString(),
  })
}

export function rollbackRelease(id: string) {
  if (!useMockApi) {
    return postData<ReleaseActionResult>(`/api/releases/${id}/rollback`)
  }
  return Promise.resolve<ReleaseActionResult>({
    releaseId: id,
    action: 'rollback',
    status: 'RUNNING',
    message: '已提交回滚任务',
    updatedAt: new Date().toISOString(),
  })
}
