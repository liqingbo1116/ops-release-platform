# 前端结构说明

## 页面

```text
pages/
  DashboardPage.vue
  EnvironmentPage.vue
  AgentPage.vue
  BaselineListPage.vue
  BaselineDetailPage.vue
  ComparePage.vue
  CreateReleasePage.vue
  ReleaseDetailPage.vue
  DeployListPage.vue
  DeployDetailPage.vue
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
```

## Store

```text
stores/
  environmentStore.ts
  agentStore.ts
  baselineStore.ts
  releaseStore.ts
  deployStore.ts
```

## API Client

```text
api/
  environments.ts
  agents.ts
  baselines.ts
  releases.ts
  deployTasks.ts
```

## 交互规则

- 差异对比页的文本搜索和状态筛选必须组合生效。
- 不可发布服务 checkbox 必须禁用。
- 发布服务选择数量必须基于当前实际勾选数量计算。
- 所有长表格必须支持横向滚动。
- 日志面板使用深色终端样式。
- Agent、环境配置、服务失败详情使用右侧抽屉。
