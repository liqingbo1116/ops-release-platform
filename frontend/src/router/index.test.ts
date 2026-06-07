import { describe, expect, it, beforeEach } from 'vitest'

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
