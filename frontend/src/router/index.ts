import { createRouter, createWebHistory } from 'vue-router'

import AgentPage from '@/pages/AgentPage.vue'
import BaselineDetailPage from '@/pages/BaselineDetailPage.vue'
import BaselineListPage from '@/pages/BaselineListPage.vue'
import ComparePage from '@/pages/ComparePage.vue'
import CreateReleasePage from '@/pages/CreateReleasePage.vue'
import DashboardPage from '@/pages/DashboardPage.vue'
import DeployDetailPage from '@/pages/DeployDetailPage.vue'
import DeployListPage from '@/pages/DeployListPage.vue'
import EnvironmentPage from '@/pages/EnvironmentPage.vue'
import ReleaseDetailPage from '@/pages/ReleaseDetailPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
  ],
})

export default router
