import { getData, useMockApi } from './client'

export type AgentTaskStatus = {
  enabled: boolean
  message?: string
  status?: {
    taskId?: string
    type?: string
    step?: string
    status?: string
    updatedAt?: string
  }
  logs?: string[]
}

export function getAgentTaskStatus(taskId: string) {
  if (!taskId || useMockApi) {
    return Promise.resolve<AgentTaskStatus>({
      enabled: false,
      message: 'agent task polling is disabled in mock mode',
      logs: [],
    })
  }
  return getData<AgentTaskStatus>(`/api/agent-tasks/${taskId}/status`)
}
