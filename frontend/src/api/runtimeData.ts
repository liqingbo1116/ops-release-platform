import { getData, type PageResult, useMockApi } from './client'
import { agentMockData } from './mockData/agent'
import { authMockData } from './mockData/auth'
import { baselineMockData } from './mockData/baseline'
import { changelogMockData } from './mockData/changelog'
import { deployMockData } from './mockData/deploy'
import { environmentMockData } from './mockData/environment'
import { releaseMockData } from './mockData/release'
import { userMockData } from './mockData/user'

type ListKey =
  | 'agents'
  | 'baselines'
  | 'deployTasks'
  | 'environments'
  | 'users'
  | 'roles'
  | 'permissions'
  | 'changelog'

type ListTargets = {
  [K in ListKey]: (items: unknown[]) => void
}

const listTargets: ListTargets = {
  agents: (items) => {
    agentMockData.agents = items as typeof agentMockData.agents
  },
  baselines: (items) => {
    baselineMockData.baselines = items as typeof baselineMockData.baselines
  },
  deployTasks: (items) => {
    deployMockData.deployTasks = items as typeof deployMockData.deployTasks
  },
  environments: (items) => {
    environmentMockData.environments = items as typeof environmentMockData.environments
  },
  users: (items) => {
    userMockData.users = items as typeof userMockData.users
  },
  roles: (items) => {
    userMockData.roles = items as typeof userMockData.roles
  },
  permissions: (items) => {
    userMockData.permissions = items as typeof userMockData.permissions
  },
  changelog: (items) => {
    changelogMockData.changelog = items as typeof changelogMockData.changelog
  },
}

async function loadList<K extends ListKey, T>(key: K, url: string) {
  const result = await getData<PageResult<T>>(url)
  listTargets[key](result.items as unknown[])
}

export async function loadRuntimeData() {
  if (useMockApi) return

  try {
    await Promise.all([
      loadList('environments', '/api/environments'),
      loadList('agents', '/api/agents'),
      loadList('baselines', '/api/baselines'),
      getData<typeof baselineMockData.baselineDetail>('/api/baselines/BL-20260607-0001').then((data) => {
        baselineMockData.baselineDetail = data
      }),
      postCompare(),
      getData<typeof releaseMockData.releaseDetail>('/api/releases/REL-20260607-031').then((data) => {
        releaseMockData.releaseDetail = data
      }),
      loadList('deployTasks', '/api/deploy-tasks'),
      getData<typeof deployMockData.deployDetail>('/api/deploy-tasks/DEP-20260607-009').then((data) => {
        deployMockData.deployDetail = data
      }),
      getData<typeof authMockData.currentUser>('/api/auth/me').then((data) => {
        authMockData.currentUser = data
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
  baselineMockData.diffResult = payload.data
}
