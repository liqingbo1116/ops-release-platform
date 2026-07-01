import { getData } from './client'

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
  probe?: AgentTaskProbeResult
}

export type AgentTaskProbeResult = {
  status?: string
  checks?: AgentTaskProbeCheck[]
}

export type AgentTaskProbeCheck = {
  component?: string
  name?: string
  status: string
  message: string
  checkedAt?: string
}

export function getAgentTaskStatus(taskId: string) {
  if (!taskId) {
    return Promise.resolve<AgentTaskStatus>({
      enabled: false,
      message: '未关联 Agent 任务',
      logs: [],
    })
  }
  return getData<AgentTaskStatus>(`/api/agent-tasks/${taskId}/status`)
}
