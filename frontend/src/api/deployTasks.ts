import { deployMockData } from './mockData/deploy'
import { getData, postData, type PageResult, useMockApi } from './client'

export async function listDeployTasks() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof deployMockData.deployTasks[number]>>('/api/deploy-tasks')
    return result.items
  }
  return Promise.resolve(deployMockData.deployTasks)
}

export type CreateDeployTaskResult = {
  id: string
  status: string
  executionMode?: string
  agentTaskId?: string
  createdAt: string
}

export type DeployStepActionResult = {
  taskId: string
  stepId: string
  action: string
  status: string
  message?: string
  updatedAt?: string
}

export function getDeployTaskDetail(id = 'DEP-20260607-009') {
  if (!useMockApi) {
    return getData<typeof deployMockData.deployDetail>(`/api/deploy-tasks/${id}`)
  }
  return Promise.resolve(deployMockData.deployDetail)
}

export function createDeployTask(body: unknown = {}) {
  if (!useMockApi) {
    return postData<CreateDeployTaskResult>('/api/deploy-tasks', body)
  }
  return Promise.resolve<CreateDeployTaskResult>({
    id: 'DEP-20260607-MOCK',
    status: 'PENDING',
    executionMode: 'AGENT',
    agentTaskId: 'DEP-20260607-MOCK',
    createdAt: new Date().toISOString(),
  })
}

export function retryDeployStep(taskId: string, stepId: string) {
  if (!useMockApi) {
    return postData<DeployStepActionResult>(`/api/deploy-tasks/${taskId}/steps/${stepId}/retry`)
  }
  return Promise.resolve<DeployStepActionResult>({
    taskId,
    stepId,
    action: 'retry',
    status: 'RUNNING',
    message: '已提交步骤重试',
    updatedAt: new Date().toISOString(),
  })
}

export function skipDeployStep(taskId: string, stepId: string) {
  if (!useMockApi) {
    return postData<DeployStepActionResult>(`/api/deploy-tasks/${taskId}/steps/${stepId}/skip`)
  }
  return Promise.resolve<DeployStepActionResult>({
    taskId,
    stepId,
    action: 'skip',
    status: 'RUNNING',
    message: '已提交步骤跳过',
    updatedAt: new Date().toISOString(),
  })
}

export function confirmDeployStep(taskId: string, stepId: string) {
  if (!useMockApi) {
    return postData<DeployStepActionResult>(`/api/deploy-tasks/${taskId}/steps/${stepId}/confirm`)
  }
  return Promise.resolve<DeployStepActionResult>({
    taskId,
    stepId,
    action: 'confirm',
    status: 'RUNNING',
    message: '已提交人工确认',
    updatedAt: new Date().toISOString(),
  })
}
