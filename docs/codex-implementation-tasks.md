# Codex 实施任务拆分

将本项目交给 VSCode + Codex 开发时，建议按下面顺序逐步执行，不要一次性要求实现全部功能。

## 总提示词

```text
你正在开发一个企业内部运维发布交付平台。请以 docs/PRD.md 为业务依据，以 design/ops-release-console-v3.html 为视觉和页面结构参考，以 docs/domain-model.md、docs/state-machine.md、docs/api-contract.md、mocks/ 为开发约束。

技术栈固定：前端 Vue 3 + Vite + TypeScript + Pinia + Vue Router + Element Plus；后端 Go + Gin + GORM；数据库 PostgreSQL；缓存与任务队列 Redis；部署使用 docker-compose。

第一阶段只需要可本地运行的 MVP。第三方系统 Jenkins、Harbor、Kubernetes、GitLab、ArgoCD、Nacos 先使用 mock adapter，不要直接写真实集成。
```

## 任务 1：初始化工程

```text
请初始化项目工程：frontend 使用 Vue 3 + Vite + TypeScript + Element Plus，backend 使用 Go + Gin。添加 docker-compose.yml，包含 frontend、backend、postgres、redis。先保证空工程能启动。
```

验收：

- `frontend` 可以 `npm run dev`
- `backend` 可以 `go run ./cmd/server`
- `docker compose up` 可以启动 postgres 和 redis

## 任务 2：前端静态页面工程化

```text
请根据 design/ops-release-console-v3.html 拆分 Vue 页面和组件。先使用 mocks/ 下的 JSON 数据，不连接后端。实现 Layout、侧边导航、顶部栏、首页、环境管理、Agent 管理、基线列表、基线详情、差异对比、创建发布单、发布详情、部署列表、部署详情。
```

重点：

- 差异筛选和搜索组合生效
- 不可发布服务禁用 checkbox
- 长表格横向滚动
- 抽屉展示环境配置、Agent 注册、服务失败详情

## 任务 3：后端 mock API

```text
请基于 docs/api-contract.md 实现 Go REST API。数据先从 mocks/ JSON 加载或在后端内置 mock repository。返回统一响应格式。
```

重点接口：

- `/api/environments`
- `/api/agents`
- `/api/baselines`
- `/api/baselines/{id}`
- `/api/baselines/{id}/compare`
- `/api/releases`
- `/api/releases/{id}`
- `/api/deploy-tasks`
- `/api/deploy-tasks/{id}`

## 任务 4：前后端联调

```text
请把前端 mock JSON 替换为后端 API 调用。保留一个 mock 模式开关，便于没有后端时前端仍可运行。
```

## 任务 5：数据库模型

```text
请根据 docs/domain-model.md 设计 PostgreSQL 表结构和 GORM model，添加数据库迁移。先支持环境、Agent、基线、发布单、部署任务、操作日志。
```

## 任务 6：任务与 Agent 模拟

```text
请使用 Redis Stream 实现平台任务队列和 mock Agent worker。创建发布单或部署任务后，mock worker 按步骤更新任务状态并追加日志。
```

## 任务 7：测试和收口

```text
请根据 docs/acceptance-criteria.md 补充前端关键交互测试和后端 API 测试，确保 MVP 验收项通过。
```
