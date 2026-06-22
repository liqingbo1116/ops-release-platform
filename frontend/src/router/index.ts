import { createRouter, createWebHistory } from 'vue-router'

const LoginPage = () => import('@/pages/LoginPage.vue')
const DashboardPage = () => import('@/pages/DashboardPage.vue')
const ProjectPage = () => import('@/pages/ProjectPage.vue')
const EnvironmentPage = () => import('@/pages/EnvironmentPage.vue')
const IntegrationResourcePage = () => import('@/pages/IntegrationResourcePage.vue')
const AgentPage = () => import('@/pages/AgentPage.vue')
const BaselineListPage = () => import('@/pages/BaselineListPage.vue')
const BaselineDetailPage = () => import('@/pages/BaselineDetailPage.vue')
const ComparePage = () => import('@/pages/ComparePage.vue')
const ReleaseListPage = () => import('@/pages/ReleaseListPage.vue')
const CreateReleasePage = () => import('@/pages/CreateReleasePage.vue')
const ReleaseDetailPage = () => import('@/pages/ReleaseDetailPage.vue')
const DeployListPage = () => import('@/pages/DeployListPage.vue')
const DeployDetailPage = () => import('@/pages/DeployDetailPage.vue')
const UserListPage = () => import('@/pages/UserListPage.vue')
const RoleListPage = () => import('@/pages/RoleListPage.vue')
const EnvironmentPermissionPage = () => import('@/pages/EnvironmentPermissionPage.vue')
const ChangelogPage = () => import('@/pages/ChangelogPage.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginPage, meta: { title: '登录', public: true, bare: true } },
    { path: '/', redirect: '/dashboard' },
    { path: '/dashboard', name: 'dashboard', component: DashboardPage, meta: { title: '首页工作台' } },
    { path: '/projects', name: 'projects', component: ProjectPage, meta: { title: '项目管理' } },
    { path: '/environments', name: 'environments', component: EnvironmentPage, meta: { title: '产品管理' } },
    { path: '/integration-resources', name: 'integration-resources', component: IntegrationResourcePage, meta: { title: '基础资源' } },
    { path: '/agents', name: 'agents', component: AgentPage, meta: { title: 'Agent 管理' } },
    { path: '/baselines', name: 'baselines', component: BaselineListPage, meta: { title: '环境基线列表' } },
    { path: '/baselines/:id', name: 'baseline-detail', component: BaselineDetailPage, meta: { title: '基线详情' } },
    { path: '/compare', name: 'compare', component: ComparePage, meta: { title: '环境差异对比' } },
    { path: '/releases', name: 'release-list', component: ReleaseListPage, meta: { title: '发布单列表' } },
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
