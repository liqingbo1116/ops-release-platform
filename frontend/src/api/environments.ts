import { getData, postData, putData, type PageResult } from './client'

export type EnvironmentInfo = {
  id: string
  name: string
  code: string
  projectId: string
  projectName: string
  productStatus: 'UNBOUND' | 'BOUND' | 'DISABLED' | string
  type: 'LOCAL' | 'PROJECT'
  deployTargetType: 'KUBERNETES' | 'DOCKER_COMPOSE'
  networkMode: 'DIRECT' | 'AGENT'
  clusterId: string
  namespace: string
  registryId: string
  registryProject: string
  privateRegistryHost: string
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
  bindingRole?: 'BUILD_SOURCE' | 'RUNTIME_TARGET'
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
  | 'projectId'
  | 'productStatus'
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

export type ProductService = {
  id: string
  productId: string
  name: string
  namespace: string
  workloadName: string
  workloadType: string
  containerName: string
  containerType: 'APP' | 'INIT' | string
  image: string
  imageRegistry: string
  imageProject: string
  imageRepository: string
  imageTag: string
  imageSource: 'PRIVATE' | 'EXTERNAL' | 'UNMATCHED_PRIVATE' | string
  privateRegistryHost?: string
  privateRegistryConfirmed?: boolean
  jenkinsJobName?: string
  jenkinsBranch?: string
  jenkinsPipelineBound?: boolean
  pipelineBoundAt?: string
  replicas: number
  readyReplicas: number
  createdAt?: string
  updatedAt?: string
  managed?: boolean
}

export type DiscoveredProductService = ProductService & {
  managed: boolean
}

function normalizeEnvironment(item: {
  id: string
  name: string
  code: string
  projectId?: string
  projectName?: string
  productStatus?: string
  type: string
  deployTargetType?: string
  networkMode: string
  clusterId?: string
  namespace?: string
  registryId?: string
  registryProject?: string
  privateRegistryHost?: string
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
    projectId: item.projectId ?? '',
    projectName: item.projectName ?? '',
    productStatus: item.productStatus ?? 'UNBOUND',
    deployTargetType: item.deployTargetType === 'DOCKER_COMPOSE' ? 'DOCKER_COMPOSE' : 'KUBERNETES',
    networkMode: item.networkMode === 'DIRECT' ? 'DIRECT' : 'AGENT',
    clusterId: item.clusterId ?? '',
    namespace: item.namespace ?? '',
    registryId: item.registryId ?? '',
    registryProject: item.registryProject ?? '',
    privateRegistryHost: item.privateRegistryHost ?? '',
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

export async function listEnvironmentServices(id: string): Promise<ProductService[]> {
  const result = await getData<PageResult<ProductService>>(`/api/environments/${id}/services?pageSize=500`)
  return result.items
}

export async function listDiscoveredEnvironmentServices(id: string): Promise<DiscoveredProductService[]> {
  const result = await getData<PageResult<DiscoveredProductService>>(
    `/api/environments/${id}/discovered-services?pageSize=500`,
  )
  return result.items
}

export async function adoptEnvironmentServices(
  id: string,
  services: DiscoveredProductService[],
): Promise<ProductService[]> {
  return postData<ProductService[]>(`/api/environments/${id}/services/adopt`, { services })
}

export async function removeEnvironmentServices(id: string, serviceIds: string[]): Promise<ProductService[]> {
  return postData<ProductService[]>(`/api/environments/${id}/services/remove`, { serviceIds })
}

export async function confirmEnvironmentServiceRegistry(
  id: string,
  privateRegistryHost: string,
): Promise<ProductService[]> {
  return postData<ProductService[]>(`/api/environments/${id}/services/confirm-registry`, { privateRegistryHost })
}

export async function bindEnvironmentServicePipeline(
  id: string,
  serviceId: string,
  payload: { jenkinsJobName: string; jenkinsBranch: string },
): Promise<ProductService> {
  return postData<ProductService>(`/api/environments/${id}/services/${serviceId}/pipeline`, payload)
}
