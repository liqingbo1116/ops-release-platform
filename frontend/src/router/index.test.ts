import { beforeEach, describe, expect, it, vi } from 'vitest'

const pageStub = { template: '<div />' }

vi.mock('@/pages/LoginPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/DashboardPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/EnvironmentPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/AgentPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/BaselineListPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/BaselineDetailPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/ComparePage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/ReleaseListPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/CreateReleasePage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/ReleaseDetailPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/DeployListPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/DeployDetailPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/UserListPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/RoleListPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/EnvironmentPermissionPage.vue', () => ({ default: pageStub }))
vi.mock('@/pages/ChangelogPage.vue', () => ({ default: pageStub }))

import router from './index'

describe('router auth guard', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('redirects protected pages to login when token is missing', async () => {
    await router.push('/dashboard')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
    expect(router.currentRoute.value.query.redirect).toBe('/dashboard')
  })

  it('allows protected pages when token exists', async () => {
    localStorage.setItem('ops-release-token', 'mock-token-admin')

    await router.push('/dashboard')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/dashboard')
  })
})
