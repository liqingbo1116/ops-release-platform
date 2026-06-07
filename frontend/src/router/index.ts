import { createRouter, createWebHistory } from 'vue-router'

import AgentPage from '@/pages/AgentPage.vue'
import BaselineDetailPage from '@/pages/BaselineDetailPage.vue'
import BaselineListPage from '@/pages/BaselineListPage.vue'
import ChangelogPage from '@/pages/ChangelogPage.vue'
import ComparePage from '@/pages/ComparePage.vue'
import CreateReleasePage from '@/pages/CreateReleasePage.vue'
import DashboardPage from '@/pages/DashboardPage.vue'
import DeployDetailPage from '@/pages/DeployDetailPage.vue'
import DeployListPage from '@/pages/DeployListPage.vue'
import EnvironmentPermissionPage from '@/pages/EnvironmentPermissionPage.vue'
import EnvironmentPage from '@/pages/EnvironmentPage.vue'
import LoginPage from '@/pages/LoginPage.vue'
import ReleaseDetailPage from '@/pages/ReleaseDetailPage.vue'
import RoleListPage from '@/pages/RoleListPage.vue'
import UserListPage from '@/pages/UserListPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginPage, meta: { title: '登录', public: true, bare: true } },
    { path: '/', redirect: '/dashboard' },
    { path: '/dashboard', name: 'dashboard', component: DashboardPage, meta: { title: '首页工作台' } },
    { path: '/environments', name: 'environments', component: EnvironmentPage, meta: { title: '环境管理' } },
    { path: '/agents', name: 'agents', component: AgentPage, meta: { title: 'Agent 管理' } },
    { path: '/baselines', name: 'baselines', component: BaselineListPage, meta: { title: '环境基线列表' } },
    { path: '/baselines/:id', name: 'baseline-detail', component: BaselineDetailPage, meta: { title: '基线详情' } },
    { path: '/compare', name: 'compare', component: ComparePage, meta: { title: '环境差异对比' } },
    { path: '/releases/create', name: 'create-release', component: CreateReleasePage, meta: { title: '创建发布单' } },
    { path: '/releases/:id', name: 'release-detail', component: ReleaseDetailPage, meta: { title: '发布详情' } },
    { path: '/deploy-tasks', name: 'deploy-list', component: DeployListPage, meta: { title: '部署任务列表' } },
    { path: '/deploy-tasks/:id', name: 'deploy-detail', component: DeployDetailPage, meta: { title: '部署任务详情' } },
    { path: '/users', name: 'users', component: UserListPage, meta: { title: '用户管理' } },
    { path: '/roles', name: 'roles', component: RoleListPage, meta: { title: '角色管理' } },
    { path: '/permissions/environments', name: 'environment-permissions', component: EnvironmentPermissionPage, meta: { title: '环境权限' } },
    { path: '/changelog', name: 'changelog', component: ChangelogPage, meta: { title: '更新日志' } },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('ops-release-token')
  if (!to.meta.public && !token) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.path === '/login' && token) {
    return { path: '/dashboard' }
  }
  return true
})

export default router
