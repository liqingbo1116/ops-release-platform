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
