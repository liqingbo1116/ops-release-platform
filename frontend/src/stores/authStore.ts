import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { getCurrentUser, login as loginApi, logout as logoutApi } from '@/api/auth'
import { authMockData } from '@/api/mockData/auth'

type CurrentUser = typeof authMockData.currentUser

const tokenKey = 'ops-release-token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(tokenKey) ?? '')
  const user = ref<CurrentUser | null>(token.value ? authMockData.currentUser : null)
  const isAuthenticated = computed(() => Boolean(token.value))

  async function login(username: string, password: string) {
    const result = await loginApi(username, password)
    token.value = result.token
    user.value = result.user
    localStorage.setItem(tokenKey, result.token)
  }

  async function loadCurrentUser() {
    if (!token.value) return
    user.value = await getCurrentUser()
  }

  async function logout() {
    await logoutApi()
    token.value = ''
    user.value = null
    localStorage.removeItem(tokenKey)
  }

  function hasPermission(permission: string) {
    const permissions = user.value?.permissions ?? []
    return permissions.includes('*') || permissions.includes(permission)
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    loadCurrentUser,
    logout,
    hasPermission,
  }
})
