import { getData, postData, putData, type PageResult } from './client'

export type EnvironmentInfo = {
  id: string
  name: string
  code: string
  type: 'LOCAL' | 'PROJECT'
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
}

export type EnvironmentPayload = Pick<
  EnvironmentInfo,
  | 'id'
  | 'name'
  | 'code'
  | 'type'
  | 'networkMode'
  | 'clusterId'
  | 'namespace'
  | 'registryId'
  | 'registryProject'
  | 'jenkinsId'
  | 'jenkinsView'
> & {
  status?: string
}

export type EnvironmentCheckResult = {
  environmentId: string
  status: string
  checkedAt: string
  checks: Array<{
    name: string
    status: string
    message: string
  }>
}

function normalizeEnvironment(item: {
  id: string
  name: string
  code: string
  type: string
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
}): EnvironmentInfo {
  return {
    ...item,
    type: item.type === 'LOCAL' ? 'LOCAL' : 'PROJECT',
    networkMode: item.networkMode === 'DIRECT' ? 'DIRECT' : 'AGENT',
    clusterId: item.clusterId ?? '',
    namespace: item.namespace ?? '',
    registryId: item.registryId ?? '',
    registryProject: item.registryProject ?? '',
    jenkinsId: item.jenkinsId ?? '',
    jenkinsView: item.jenkinsView ?? '',
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
