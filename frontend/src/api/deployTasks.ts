import { mockData } from './mockData'
import { getData, postData, type PageResult, useMockApi } from './client'

export async function listDeployTasks() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.deployTasks[number]>>('/api/deploy-tasks')
    return result.items
  }
  return Promise.resolve(mockData.deployTasks)
}

export type CreateDeployTaskResult = {
  id: string
  status: string
  createdAt: string
}

export function getDeployTaskDetail(id = 'DEP-20260607-009') {
  if (!useMockApi) {
    return getData<typeof mockData.deployDetail>(`/api/deploy-tasks/${id}`)
  }
  return Promise.resolve(mockData.deployDetail)
}

export function createDeployTask(body: unknown = {}) {
  if (!useMockApi) {
    return postData<CreateDeployTaskResult>('/api/deploy-tasks', body)
  }
  return Promise.resolve<CreateDeployTaskResult>({
    id: 'DEP-20260607-MOCK',
    status: 'PENDING',
    createdAt: new Date().toISOString(),
  })
}
