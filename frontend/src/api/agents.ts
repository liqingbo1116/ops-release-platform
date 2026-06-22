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
  lastHeartbeatAt: string
  currentTaskId: string | null
}

export type AgentRegisterToken = {
  platformUrl: string
  token: string
  expiresAt: string
  installCommand: string
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
