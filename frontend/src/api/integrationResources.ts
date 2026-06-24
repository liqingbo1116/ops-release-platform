import { getData, postData, putData, type PageResult } from './client'
import type { JenkinsPipeline } from './releases'

export type KubernetesCluster = {
  id: string
  name: string
  apiServer: string
  context: string
  status: string
  lastCheckAt: string
  probeMessage: string
  namespaces: string[]
}

export type HarborRegistry = {
  id: string
  name: string
  url: string
  registryHost: string
  scheme: 'http' | 'https'
  username: string
  insecureSkipTLSVerify: boolean
  status: string
  lastCheckAt: string
  probeMessage: string
  projects: string[]
}

export type JenkinsInstance = {
  id: string
  name: string
  url: string
  username: string
  insecureSkipTLSVerify: boolean
  status: string
  lastCheckAt: string
  probeMessage: string
  views: string[]
  jobs: string[]
  pipelines: JenkinsPipeline[]
}

export type IntegrationResource = KubernetesCluster | HarborRegistry | JenkinsInstance
export type IntegrationResourceKind = 'kubernetes' | 'harbor' | 'jenkins'

export type KubernetesClusterPayload = Pick<KubernetesCluster, 'id' | 'name'> & {
  apiServer?: string
  context?: string
  kubeconfig?: string
}
export type HarborRegistryPayload = Pick<
  HarborRegistry,
  'id' | 'name' | 'url' | 'scheme' | 'username' | 'insecureSkipTLSVerify'
> & {
  password?: string
}
export type JenkinsInstancePayload = Pick<JenkinsInstance, 'id' | 'name' | 'url' | 'username' | 'insecureSkipTLSVerify'> & {
  token?: string
}

function normalizeKubernetesCluster(item: KubernetesCluster): KubernetesCluster {
  return {
    ...item,
    status: item.status || 'UNKNOWN',
    lastCheckAt: item.lastCheckAt ?? '',
    probeMessage: item.probeMessage ?? '',
    context: item.context ?? '',
    namespaces: item.namespaces ?? [],
  }
}

function normalizeHarborRegistry(item: HarborRegistry): HarborRegistry {
  return {
    ...item,
    scheme: item.scheme === 'https' ? 'https' : 'http',
    username: item.username ?? '',
    insecureSkipTLSVerify: item.insecureSkipTLSVerify ?? false,
    status: item.status || 'UNKNOWN',
    lastCheckAt: item.lastCheckAt ?? '',
    probeMessage: item.probeMessage ?? '',
    projects: item.projects ?? [],
  }
}

function normalizeJenkinsInstance(item: JenkinsInstance): JenkinsInstance {
  return {
    ...item,
    username: item.username ?? '',
    insecureSkipTLSVerify: item.insecureSkipTLSVerify ?? false,
    status: item.status || 'UNKNOWN',
    lastCheckAt: item.lastCheckAt ?? '',
    probeMessage: item.probeMessage ?? '',
    views: item.views ?? [],
    jobs: item.jobs ?? [],
    pipelines: item.pipelines ?? [],
  }
}

export async function listKubernetesClusters(): Promise<KubernetesCluster[]> {
  const result = await getData<PageResult<KubernetesCluster>>('/api/kubernetes-clusters')
  return result.items.map(normalizeKubernetesCluster)
}

export async function createKubernetesCluster(payload: KubernetesClusterPayload): Promise<KubernetesCluster> {
  return normalizeKubernetesCluster(await postData<KubernetesCluster>('/api/kubernetes-clusters', payload))
}

export async function updateKubernetesCluster(
  id: string,
  payload: Partial<KubernetesClusterPayload>,
): Promise<KubernetesCluster> {
  return normalizeKubernetesCluster(await putData<KubernetesCluster>(`/api/kubernetes-clusters/${id}`, payload))
}

export async function testKubernetesCluster(id: string): Promise<KubernetesCluster> {
  return normalizeKubernetesCluster(await postData<KubernetesCluster>(`/api/kubernetes-clusters/${id}/test`))
}

export async function refreshKubernetesCluster(id: string): Promise<KubernetesCluster> {
  return normalizeKubernetesCluster(await postData<KubernetesCluster>(`/api/kubernetes-clusters/${id}/refresh`))
}

export async function listHarborRegistries(): Promise<HarborRegistry[]> {
  const result = await getData<PageResult<HarborRegistry>>('/api/harbor-registries')
  return result.items.map(normalizeHarborRegistry)
}

export async function createHarborRegistry(payload: HarborRegistryPayload): Promise<HarborRegistry> {
  return normalizeHarborRegistry(await postData<HarborRegistry>('/api/harbor-registries', payload))
}

export async function updateHarborRegistry(id: string, payload: Partial<HarborRegistryPayload>): Promise<HarborRegistry> {
  return normalizeHarborRegistry(await putData<HarborRegistry>(`/api/harbor-registries/${id}`, payload))
}

export async function testHarborRegistry(id: string): Promise<HarborRegistry> {
  return normalizeHarborRegistry(await postData<HarborRegistry>(`/api/harbor-registries/${id}/test`))
}

export async function refreshHarborRegistry(id: string): Promise<HarborRegistry> {
  return normalizeHarborRegistry(await postData<HarborRegistry>(`/api/harbor-registries/${id}/refresh`))
}

export async function listJenkinsInstances(): Promise<JenkinsInstance[]> {
  const result = await getData<PageResult<JenkinsInstance>>('/api/jenkins-instances')
  return result.items.map(normalizeJenkinsInstance)
}

export async function createJenkinsInstance(payload: JenkinsInstancePayload): Promise<JenkinsInstance> {
  return normalizeJenkinsInstance(await postData<JenkinsInstance>('/api/jenkins-instances', payload))
}

export async function updateJenkinsInstance(
  id: string,
  payload: Partial<JenkinsInstancePayload>,
): Promise<JenkinsInstance> {
  return normalizeJenkinsInstance(await putData<JenkinsInstance>(`/api/jenkins-instances/${id}`, payload))
}

export async function testJenkinsInstance(id: string): Promise<JenkinsInstance> {
  return normalizeJenkinsInstance(await postData<JenkinsInstance>(`/api/jenkins-instances/${id}/test`))
}

export async function refreshJenkinsInstance(id: string): Promise<JenkinsInstance> {
  return normalizeJenkinsInstance(await postData<JenkinsInstance>(`/api/jenkins-instances/${id}/refresh`))
}
