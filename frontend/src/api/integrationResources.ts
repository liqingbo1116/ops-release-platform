import { getData, postData, putData, type PageResult } from './client'

export type KubernetesCluster = {
  id: string
  name: string
  apiServer: string
  credentialRef: string
  status: string
  lastCheckAt: string
}

export type HarborRegistry = {
  id: string
  name: string
  url: string
  credentialRef: string
  status: string
  lastCheckAt: string
}

export type JenkinsInstance = {
  id: string
  name: string
  url: string
  credentialRef: string
  status: string
  lastCheckAt: string
}

export type IntegrationResource = KubernetesCluster | HarborRegistry | JenkinsInstance
export type IntegrationResourceKind = 'kubernetes' | 'harbor' | 'jenkins'

export type KubernetesClusterPayload = Omit<KubernetesCluster, 'lastCheckAt'> & { lastCheckAt?: string }
export type HarborRegistryPayload = Omit<HarborRegistry, 'lastCheckAt'> & { lastCheckAt?: string }
export type JenkinsInstancePayload = Omit<JenkinsInstance, 'lastCheckAt'> & { lastCheckAt?: string }

function normalizeResource<T extends IntegrationResource>(item: T): T {
  return {
    ...item,
    credentialRef: item.credentialRef ?? '',
    status: item.status || 'UNKNOWN',
    lastCheckAt: item.lastCheckAt ?? '',
  }
}

export async function listKubernetesClusters(): Promise<KubernetesCluster[]> {
  const result = await getData<PageResult<KubernetesCluster>>('/api/kubernetes-clusters')
  return result.items.map(normalizeResource)
}

export async function createKubernetesCluster(payload: KubernetesClusterPayload): Promise<KubernetesCluster> {
  return normalizeResource(await postData<KubernetesCluster>('/api/kubernetes-clusters', payload))
}

export async function updateKubernetesCluster(
  id: string,
  payload: Partial<KubernetesClusterPayload>,
): Promise<KubernetesCluster> {
  return normalizeResource(await putData<KubernetesCluster>(`/api/kubernetes-clusters/${id}`, payload))
}

export async function listHarborRegistries(): Promise<HarborRegistry[]> {
  const result = await getData<PageResult<HarborRegistry>>('/api/harbor-registries')
  return result.items.map(normalizeResource)
}

export async function createHarborRegistry(payload: HarborRegistryPayload): Promise<HarborRegistry> {
  return normalizeResource(await postData<HarborRegistry>('/api/harbor-registries', payload))
}

export async function updateHarborRegistry(id: string, payload: Partial<HarborRegistryPayload>): Promise<HarborRegistry> {
  return normalizeResource(await putData<HarborRegistry>(`/api/harbor-registries/${id}`, payload))
}

export async function listJenkinsInstances(): Promise<JenkinsInstance[]> {
  const result = await getData<PageResult<JenkinsInstance>>('/api/jenkins-instances')
  return result.items.map(normalizeResource)
}

export async function createJenkinsInstance(payload: JenkinsInstancePayload): Promise<JenkinsInstance> {
  return normalizeResource(await postData<JenkinsInstance>('/api/jenkins-instances', payload))
}

export async function updateJenkinsInstance(
  id: string,
  payload: Partial<JenkinsInstancePayload>,
): Promise<JenkinsInstance> {
  return normalizeResource(await putData<JenkinsInstance>(`/api/jenkins-instances/${id}`, payload))
}
