# 前端结构说明

## 页面

```text
pages/
  DashboardPage.vue
  LoginPage.vue
  EnvironmentPage.vue
  AgentPage.vue
  BaselineListPage.vue
  BaselineDetailPage.vue
  ComparePage.vue
  CreateReleasePage.vue
  ReleaseDetailPage.vue
  DeployListPage.vue
  DeployDetailPage.vue
  UserListPage.vue
  RoleListPage.vue
  EnvironmentPermissionPage.vue
  ChangelogPage.vue
```

## 组件

```text
components/
  AppLayout.vue
  SideNav.vue
  TopBar.vue
  StatusTag.vue
  MetricCard.vue
  DataTable.vue
  StepTimeline.vue
  LogTerminal.vue
  EnvironmentConfigDrawer.vue
  AgentRegisterDrawer.vue
  ServiceDiffTable.vue
  ReleaseRiskPanel.vue
  DeployStepPanel.vue
  PermissionButton.vue
  ChangelogTimeline.vue
```

## Store

```text
stores/
  environmentStore.ts
  agentStore.ts
  baselineStore.ts
  releaseStore.ts
  deployStore.ts
  authStore.ts
  userStore.ts
  changelogStore.ts
```

## API Client

```text
api/
  environments.ts
  agents.ts
  baselines.ts
  releases.ts
  deployTasks.ts
  auth.ts
  users.ts
  changelog.ts
```

## 交互规则

- 差异对比页的文本搜索和状态筛选必须组合生效。
- 不可发布服务 checkbox 必须禁用。
- 发布服务选择数量必须基于当前实际勾选数量计算。
- 所有长表格必须支持横向滚动。
- 日志面板使用深色终端样式。
- Agent、环境配置、服务失败详情使用右侧抽屉。
- 未登录访问业务路由时跳转登录页。
- 登录后顶部栏展示当前用户、角色和退出入口。
- 用户、角色、环境权限页面必须使用真实后端数据；未接入 SSO 前使用平台后端认证接口。
- 更新日志页面按版本号倒序展示，支持版本号、类型和关键词筛选。
