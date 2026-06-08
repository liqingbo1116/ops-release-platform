import { releaseMockData } from './mockData/release'
import { getData, postData, type PageResult, useMockApi } from './client'

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
  buildId?: string
  buildStatus?: string
  createdAt: string
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
    status: 'PENDING_CONFIRM',
    createdAt: new Date().toISOString(),
  })
}
