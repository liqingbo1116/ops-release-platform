import { getData, postData, putData, type PageResult } from './client'

export type EnvironmentInfo = {
  id: string
  name: string
  code: string
  type: 'LOCAL' | 'PROJECT'
  deployTargetType: 'KUBERNETES' | 'DOCKER_COMPOSE'
  networkMode: 'DIRECT' | 'AGENT'
  clusterId: string
  namespace: string
  registryId: string
  registryProject: string
  jenkinsId: string
  jenkinsView: string
  status: string
  agentStatus: string
  lastCheckAt: string
  bindings: EnvironmentResourceBinding[]
}

export type EnvironmentResourceBinding = {
  id?: string
  environmentId?: string
  resourceType: 'K8S' | 'HARBOR' | 'JENKINS'
  resourceId: string
  scopeType: 'NAMESPACE' | 'PROJECT' | 'VIEW'
  scopeValue: string
  isDefault: boolean
}

export type EnvironmentPayload = Pick<
  EnvironmentInfo,
  | 'id'
  | 'name'
  | 'code'
  | 'type'
  | 'deployTargetType'
  | 'networkMode'
  | 'clusterId'
  | 'namespace'
  | 'registryId'
  | 'registryProject'
  | 'jenkinsId'
  | 'jenkinsView'
  | 'bindings'
> & {
  status?: string
}

export type EnvironmentCheckResult = {
  environmentId: string
  status: string
  checkedAt: string
  checks: Array<{
    component?: string
    name?: string
    status: string
    message: string
  }>
}

export type EnvironmentProbeResult = {
  taskId: string
  agentId: string
  environmentId: string
  status: string
  message: string
}

function normalizeEnvironment(item: {
  id: string
  name: string
  code: string
  type: string
  deployTargetType?: string
  networkMode: string
  clusterId?: string
  namespace?: string
  registryId?: string
  registryProject?: string
  jenkinsId?: string
  jenkinsView?: string
  status: string
  agentStatus: string
  lastCheckAt: string
  bindings?: EnvironmentResourceBinding[]
}): EnvironmentInfo {
  return {
    ...item,
    type: item.type === 'LOCAL' ? 'LOCAL' : 'PROJECT',
    deployTargetType: item.deployTargetType === 'DOCKER_COMPOSE' ? 'DOCKER_COMPOSE' : 'KUBERNETES',
    networkMode: item.networkMode === 'DIRECT' ? 'DIRECT' : 'AGENT',
    clusterId: item.clusterId ?? '',
    namespace: item.namespace ?? '',
    registryId: item.registryId ?? '',
    registryProject: item.registryProject ?? '',
    jenkinsId: item.jenkinsId ?? '',
    jenkinsView: item.jenkinsView ?? '',
    bindings: item.bindings ?? [],
  }
}

export async function listEnvironments(): Promise<EnvironmentInfo[]> {
  const result = await getData<PageResult<EnvironmentInfo>>('/api/environments')
  return result.items.map(normalizeEnvironment)
}

export async function createEnvironment(payload: EnvironmentPayload): Promise<EnvironmentInfo> {
  return normalizeEnvironment(await postData<EnvironmentInfo>('/api/environments', payload))
}

export async function updateEnvironment(id: string, payload: Partial<EnvironmentPayload>): Promise<EnvironmentInfo> {
  return normalizeEnvironment(await putData<EnvironmentInfo>(`/api/environments/${id}`, payload))
}

export async function checkEnvironment(id: string): Promise<EnvironmentCheckResult> {
  return postData<EnvironmentCheckResult>(`/api/environments/${id}/check`)
}

export async function probeEnvironment(id: string): Promise<EnvironmentProbeResult> {
  return postData<EnvironmentProbeResult>(`/api/environments/${id}/remote-probe`)
}
