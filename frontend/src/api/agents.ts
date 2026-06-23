import { getData, postData, type PageResult } from './client'

export type AgentInfo = {
  id: string
  name: string
  environmentId: string
  environmentName: string
  version: string
  status: 'ONLINE' | 'OFFLINE' | 'BUSY' | string
  claimStatus: 'PENDING_CLAIM' | 'CLAIMED' | string
  capabilities: string[]
  runtimeStatus?: AgentRuntimeStatus
  lastHeartbeatAt: string
  currentTaskId: string | null
}

export type AgentRuntimeStatus = {
  kubernetes?: AgentRuntimeComponentStatus
  harbor?: AgentRuntimeComponentStatus
}

export type AgentRuntimeComponentStatus = {
  status: 'HEALTHY' | 'UNHEALTHY' | 'UNKNOWN' | string
  message: string
  updatedAt: string
  items: string[]
  workloads?: RuntimeWorkload[]
}

export type RuntimeWorkload = {
  namespace: string
  name: string
  type: string
  replicas: number
  readyReplicas: number
  containers: RuntimeContainer[]
}

export type RuntimeContainer = {
  name: string
  type: 'APP' | 'INIT' | string
  image: string
}

export type AgentRegisterToken = {
  agentId: string
  platformUrl: string
  token: string
  expiresAt: string
  configText?: string
  installCommand?: string
}

export async function listAgents(): Promise<AgentInfo[]> {
  const result = await getData<PageResult<AgentInfo>>('/api/agents')
  return result.items
}

export async function createAgentRegisterToken(agentId: string, ttlMinutes = 60): Promise<AgentRegisterToken> {
  return postData<AgentRegisterToken>('/api/agents/register-token', {
    agentId,
    ttlMinutes,
  })
}

export async function claimAgent(agentId: string, environmentId: string): Promise<AgentInfo> {
  return postData<AgentInfo>(`/api/agents/${agentId}/claim`, { environmentId })
}
