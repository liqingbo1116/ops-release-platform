import { getData, type PageResult, useMockApi } from './client'
import { mockData } from './mockData'

type ListKey =
  | 'agents'
  | 'baselines'
  | 'deployTasks'
  | 'environments'
  | 'users'
  | 'roles'
  | 'permissions'
  | 'changelog'

async function loadList<T>(key: ListKey, url: string) {
  const result = await getData<PageResult<T>>(url)
  ;(mockData[key] as T[]) = result.items
}

export async function loadRuntimeData() {
  if (useMockApi) return

  try {
    await Promise.all([
      loadList('environments', '/api/environments'),
      loadList('agents', '/api/agents'),
      loadList('baselines', '/api/baselines'),
      getData<typeof mockData.baselineDetail>('/api/baselines/BL-20260607-0001').then((data) => {
        mockData.baselineDetail = data
      }),
      postCompare(),
      getData<typeof mockData.releaseDetail>('/api/releases/REL-20260607-031').then((data) => {
        mockData.releaseDetail = data
      }),
      loadList('deployTasks', '/api/deploy-tasks'),
      getData<typeof mockData.deployDetail>('/api/deploy-tasks/DEP-20260607-009').then((data) => {
        mockData.deployDetail = data
      }),
      getData<typeof mockData.currentUser>('/api/auth/me').then((data) => {
        mockData.currentUser = data
      }),
      loadList('users', '/api/users'),
      loadList('roles', '/api/roles'),
      loadList('permissions', '/api/permissions'),
      loadList('changelog', '/api/changelog'),
    ])
  } catch (error) {
    console.warn('Backend API unavailable, falling back to local mock data.', error)
  }
}

async function postCompare() {
  const response = await fetch(`${import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:8080'}/api/baselines/BL-20260607-0001/compare`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
  if (!response.ok) throw new Error(`compare failed: ${response.status}`)
  const payload = await response.json()
  mockData.diffResult = payload.data
}
