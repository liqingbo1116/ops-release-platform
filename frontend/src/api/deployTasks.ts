import { getData, postData, type PageResult } from './client'

export type DeployTask = {
  id: string
  type?: string
  releaseId?: string
  productName?: string
  targetEnvironmentName: string
  sourceBaselineId?: string
  source?: string
  status: string
  progress: number
  serviceNames?: string[]
  missingServices?: string[]
  missingServiceCount?: number
  currentStep?: string
  agentName?: string
  agentTaskId?: string
  nextAction?: string
  createdAt?: string
}

export type DeployDetail = DeployTask & {
  agentTaskId?: string
  steps: Array<{
    id?: string
    order?: number
    name: string
    type: string
    status: string
    message?: string
    startedAt?: string
    finishedAt?: string
  }>
  logs?: string[]
  actionRecords?: Array<{
    occurredAt: string
    action: string
    operator: string
    status: string
    message?: string
  }>
  auditSummary?: {
    operator?: string
    targetEnvironmentName?: string
    affectedServices?: string[]
    result?: string
    failedStep?: string
    lastAction?: string
    lastActionAt?: string
  }
}

export async function listDeployTasks() {
  const result = await getData<PageResult<DeployTask>>('/api/deploy-tasks')
  return result.items
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

export function getDeployTaskDetail(id: string) {
  return getData<DeployDetail>(`/api/deploy-tasks/${id}`)
}

export function createDeployTask(body: unknown = {}) {
  return postData<CreateDeployTaskResult>('/api/deploy-tasks', body)
}

export function retryDeployStep(taskId: string, stepId: string) {
  return postData<DeployStepActionResult>(`/api/deploy-tasks/${taskId}/steps/${stepId}/retry`)
}

export function skipDeployStep(taskId: string, stepId: string) {
  return postData<DeployStepActionResult>(`/api/deploy-tasks/${taskId}/steps/${stepId}/skip`)
}

export function confirmDeployStep(taskId: string, stepId: string) {
  return postData<DeployStepActionResult>(`/api/deploy-tasks/${taskId}/steps/${stepId}/confirm`)
}
