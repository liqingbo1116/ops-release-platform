import { mockData } from './mockData'
import { getData, postData, useMockApi } from './client'

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

export function getReleaseDetail(id = 'REL-20260607-031') {
  if (!useMockApi) {
    return getData<typeof mockData.releaseDetail>(`/api/releases/${id}`)
  }
  return Promise.resolve(mockData.releaseDetail)
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
