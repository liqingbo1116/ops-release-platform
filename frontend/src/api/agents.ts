import { getData, postData, type PageResult } from './client'

export type AgentInfo = {
  id: string
  name: string
  environmentId: string
  environmentName: string
  version: string
  status: 'ONLINE' | 'OFFLINE' | 'BUSY' | string
  capabilities: string[]
  lastHeartbeatAt: string
  currentTaskId: string | null
}

export type AgentRegisterToken = {
  token: string
  expiresAt: string
  installCommand: string
}

export async function listAgents(): Promise<AgentInfo[]> {
  const result = await getData<PageResult<AgentInfo>>('/api/agents')
  return result.items
}

export async function createAgentRegisterToken(
  environmentId: string,
  agentId: string,
  ttlMinutes = 60,
): Promise<AgentRegisterToken> {
  return postData<AgentRegisterToken>('/api/agents/register-token', {
    agentId,
    environmentId,
    ttlMinutes,
  })
}
