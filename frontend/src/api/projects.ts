import { getData, postData, putData, type PageResult } from './client'

export type ProjectInfo = {
  id: string
  name: string
  code: string
  description: string
  status: 'ACTIVE' | 'DISABLED' | string
  productCount: number
  createdAt: string
}

export type ProjectPayload = Pick<ProjectInfo, 'id' | 'name' | 'code' | 'description' | 'status'>

function normalizeProject(item: Partial<ProjectInfo> & { id: string; name: string; code: string }): ProjectInfo {
  return {
    id: item.id,
    name: item.name,
    code: item.code,
    description: item.description ?? '',
    status: item.status ?? 'ACTIVE',
    productCount: item.productCount ?? 0,
    createdAt: item.createdAt ?? '',
  }
}

export async function listProjects(): Promise<ProjectInfo[]> {
  const result = await getData<PageResult<ProjectInfo>>('/api/projects')
  return result.items.map(normalizeProject)
}

export async function getProject(id: string): Promise<ProjectInfo> {
  return normalizeProject(await getData<ProjectInfo>(`/api/projects/${id}`))
}

export async function createProject(payload: ProjectPayload): Promise<ProjectInfo> {
  return normalizeProject(await postData<ProjectInfo>('/api/projects', payload))
}

export async function updateProject(id: string, payload: Partial<ProjectPayload>): Promise<ProjectInfo> {
  return normalizeProject(await putData<ProjectInfo>(`/api/projects/${id}`, payload))
}
