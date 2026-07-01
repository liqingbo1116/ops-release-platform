import { getData, getDataWithParams, postData, type PageResult } from './client'

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
    jobUrl?: string
    branch?: string
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

export type ReleaseOrder = {
  id: string
  type: string
  sourceBaselineId?: string
  releaseSource?: 'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE' | string
  executionMode?: string
  buildId?: string
  buildStatus?: string
  buildUrl?: string
  imageRepository?: string
  imageTag?: string
  imageDigest?: string
  targetEnvironmentName: string
  status: string
  progress: number
  agentName: string
  serviceIds?: string[]
  serviceNames?: string[]
  createdAt?: string
}

export type ReleaseStep = {
  name: string
  status: string
  message?: string
  startedAt?: string
  finishedAt?: string
}

export type ReleaseFailure = {
  serviceId?: string
  serviceName: string
  reason: string
  suggestion: string
}

export type ReleaseActionRecord = {
  occurredAt: string
  action: string
  operator: string
  status: string
  message?: string
}

export type ReleaseReport = {
  generatedAt: string
  operator: string
  successServiceCount: number
  failedServiceCount: number
  manualConfirmCount: number
  rollbackRecommended: boolean
  summary: string
}

export type ReleaseAuditSummary = {
  operator?: string
  targetEnvironmentName?: string
  affectedServices?: string[]
  result?: string
  failedStep?: string
  lastAction?: string
  lastActionAt?: string
}

export type ReleaseDetail = {
  id: string
  type: string
  sourceBaselineId?: string
  releaseSource?: 'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE' | string
  executionMode?: string
  buildId?: string
  buildStatus?: string
  buildUrl?: string
  imageRepository?: string
  imageTag?: string
  imageDigest?: string
  targetEnvironmentName: string
  status: string
  progress: number
  agentName: string
  agentTaskId?: string
  steps: ReleaseStep[]
  failures: ReleaseFailure[]
  actionRecords?: ReleaseActionRecord[]
  report?: ReleaseReport
  auditSummary?: ReleaseAuditSummary
  serviceIds?: string[]
  serviceNames?: string[]
  jenkinsId?: string
  jenkinsJobName?: string
  jenkinsJobUrl?: string
  logs?: string[]
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
  imageRegistry: string
  imageProject: string
  imageRepository: string
  imageTag: string
  imageSource: string
  privateRegistryHost?: string
  privateRegistryConfirmed: boolean
  jenkinsJobName?: string
  jenkinsJobUrl?: string
  jenkinsBranch?: string
  jenkinsPipelineBound?: boolean
  pipelineBoundAt?: string
  tags: ReleaseImageTag[]
  publishable: boolean
  message?: string
}

export type JenkinsPipelineParameter = {
  name: string
  type: string
  defaultValue?: string
  description?: string
  required: boolean
}

export type JenkinsPipeline = {
  name: string
  view?: string
  viewUrl?: string
  url?: string
  parameters?: JenkinsPipelineParameter[]
}

export type ReleaseSource = {
  environmentId: string
  services: ReleaseSourceService[]
  jenkinsJobs: string[]
  jenkinsPipelines: JenkinsPipeline[]
}

export type ReleaseActionResult = {
  releaseId: string
  action: string
  status: string
  message?: string
  updatedAt?: string
}

export type ListReleaseSourceOptions = {
  keyword?: string
  serviceId?: string
  includeTags?: boolean
}

export function listReleaseSources(environmentId: string, options: string | ListReleaseSourceOptions = '') {
  const params =
    typeof options === 'string'
      ? { environmentId, keyword: options }
      : {
          environmentId,
          keyword: options.keyword ?? '',
          serviceId: options.serviceId ?? '',
          includeTags: options.includeTags,
        }
  return getDataWithParams<ReleaseSource>('/api/release-sources', params)
}

export function listReleases(): Promise<ReleaseOrder[]> {
  return getData<PageResult<ReleaseOrder>>('/api/releases').then((result) => result.items)
}

export function listServiceReleases(productId: string, serviceId: string) {
  return getData<PageResult<ReleaseOrder>>(`/api/environments/${productId}/services/${serviceId}/releases`).then(
    (result) => result.items,
  )
}

export function getReleaseDetail(id: string) {
  return getData<ReleaseDetail>(`/api/releases/${id}`)
}

export function createRelease(body: CreateReleaseRequest) {
  return postData<CreateReleaseResult>('/api/releases', body)
}

export function retryRelease(id: string) {
  return postData<ReleaseActionResult>(`/api/releases/${id}/retry`)
}

export function rollbackRelease(id: string) {
  return postData<ReleaseActionResult>(`/api/releases/${id}/rollback`)
}
