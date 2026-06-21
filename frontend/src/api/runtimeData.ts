import { getData, type PageResult, useMockApi } from './client'
import { agentMockData } from './mockData/agent'
import { authMockData } from './mockData/auth'
import { baselineMockData } from './mockData/baseline'
import { changelogMockData } from './mockData/changelog'
import { deployMockData } from './mockData/deploy'
import { environmentMockData } from './mockData/environment'
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
      loadList('deployTasks', '/api/deploy-tasks'),
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
