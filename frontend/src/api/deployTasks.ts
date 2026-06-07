import { mockData } from './mockData'
import { getData, type PageResult, useMockApi } from './client'

export async function listDeployTasks() {
  if (!useMockApi) {
    const result = await getData<PageResult<typeof mockData.deployTasks[number]>>('/api/deploy-tasks')
    return result.items
  }
  return Promise.resolve(mockData.deployTasks)
}

export function getDeployTaskDetail() {
  if (!useMockApi) {
    return getData<typeof mockData.deployDetail>('/api/deploy-tasks/DEP-20260607-009')
  }
  return Promise.resolve(mockData.deployDetail)
}
