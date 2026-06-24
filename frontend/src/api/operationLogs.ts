import { getDataWithParams, type PageResult } from './client'

export type OperationLog = {
  id: string
  operatorId: string
  operatorName: string
  action: string
  resourceType: string
  resourceId: string
  resourceName?: string
  projectId?: string
  projectName?: string
  environmentId?: string
  productName?: string
  taskId?: string
  namespace?: string
  workloadType?: string
  workloadName?: string
  containerName?: string
  containerType?: string
  result: string
  detail: string
  createdAt: string
}

export type OperationLogQuery = {
  keyword?: string
  environmentId?: string
  resourceType?: string
}

export async function listOperationLogs(params: OperationLogQuery = {}): Promise<OperationLog[]> {
  const result = await getDataWithParams<PageResult<OperationLog>>('/api/operation-logs', {
    pageSize: 100,
    ...params,
  })
  return result.items
}
