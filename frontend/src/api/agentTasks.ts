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

function resolveMockTaskType(taskId: string) {
  return taskId.startsWith('DEP-') || taskId.includes('-DEP-') ? 'deploy' : 'release'
}

function resolveMockTaskStep(taskId: string) {
  return resolveMockTaskType(taskId) === 'deploy' ? '创建 workload' : '同步镜像并更新 tag'
}

export function getAgentTaskStatus(taskId: string) {
  if (!taskId) {
    return Promise.resolve<AgentTaskStatus>({
      enabled: false,
      message: '未关联 Agent 任务',
      logs: [],
    })
  }
  if (useMockApi) {
    const taskType = resolveMockTaskType(taskId)
    const step = resolveMockTaskStep(taskId)
    return Promise.resolve<AgentTaskStatus>({
      enabled: true,
      message: 'mock Agent task status',
      status: {
        taskId,
        type: taskType,
        step,
        status: taskType === 'deploy' ? 'RUNNING' : 'WAITING_CONFIRM',
        updatedAt: new Date().toISOString(),
      },
      logs: [
        `[INFO] mock agent accepted task ${taskId}`,
        `[INFO] ${step}`,
        taskType === 'deploy' ? '[INFO] health check is running' : '[WARN] waiting for manual release confirmation',
      ],
    })
  }
  return getData<AgentTaskStatus>(`/api/agent-tasks/${taskId}/status`)
}
