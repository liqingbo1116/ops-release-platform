# Codex 实施任务拆分

将本项目交给 VSCode + Codex 开发时，建议按下面顺序逐步执行，不要一次性要求实现全部功能。

## 当前执行原则

- 先以实现 V1 功能闭环为主，不以优化为主。
- V1 最低目标不是“页面齐全”，而是“至少支持远程项目环境的部署与管理”。
- 当研发即将触达 Jenkins、Harbor、Kubernetes、真实 Agent 联调时，必须提前说明所需环境和准备项，由环境提供方先准备，再继续真实集成开发。
- 后续默认研发顺序应围绕这条主链路推进：
  1. 运行态采集
  2. 基线生成与锁定
  3. 差异对比分类
  4. 发布任务创建与执行跟踪
  5. 部署任务创建与执行跟踪
  6. Agent 心跳、拉任务、日志回传、结果回传
  7. 审计与权限补齐
- 包体优化、构建 warning 清理、UI 打磨、纯重构清理都放在这条主链路之后，除非已经阻塞功能交付。

## 外部环境准备规则

- 允许先用 mock adapter 推进功能闭环，但不能默认真实环境“之后再说”。
- 只要某一步开始需要真实系统联调，就要提前列出需要的组件与最小准备条件。
- V1 默认 Agent 部署方案固定为：
  - Agent 运行在独立 Linux 主机
  - 使用 `docker compose` 部署
  - Agent 操作的目标环境是远程 Kubernetes 集群
  - 不把 Agent 自身先部署进 Kubernetes 作为 V1 前置条件
- 默认需要重点提前确认的外部组件：
  - Jenkins：测试 Job、凭证接入方式、可用视图或命名规则
  - Harbor/Registry：测试仓库、镜像推送与拉取权限、测试 tag 策略
  - Kubernetes：测试集群、namespace、服务账号或 kubeconfig 准备方式
  - Agent 运行节点：可部署位置、网络连通性、日志与结果回传路径
- 文档和 TODO 中应把这类准备要求写在对应研发阶段前，而不是放在最终联调时再补充。

## V1 后续执行顺序与环境门槛

1. 发布/部署详情闭环
   - 补 `agentTaskId` 关联、状态刷新、日志、失败原因
   - 用户视角：
     - 进入发布详情页可看到状态、步骤、日志、执行记录、发布报告
     - 进入部署详情页可看到状态、步骤、日志、执行记录
     - 失败或阻塞时可操作重试、跳过、人工确认、回滚
   - 若只做 mock，不要求真实外部环境
   - 若要提前真实联调，先准备 Agent `docker compose` 宿主机和平台连通性
2. Agent 协议闭环
   - 心跳、拉任务、步骤状态、日志、结果回传
   - 用户视角：
     - Agent 管理页可看到注册、在线状态、最近心跳
     - 发布/部署任务可被 Agent 拉取执行
     - 详情页状态和日志来自真实 Agent 回传
   - 进入前先准备 Linux Agent 宿主机、`docker compose`、平台连通性、测试服务
3. 真实发布链路联调
   - 用户视角：
     - 创建发布时可选择 Jenkins Job 或 Harbor 镜像 tag
     - 用户不再手工查项目 Harbor 或手工改 tag
   - 进入前先准备 Jenkins、Harbor/Registry、Agent 到两者的访问能力
   - 同时准备最小联调样本：
     - 测试服务源码仓库
     - `Dockerfile`
     - Jenkinsfile 或构建脚本
     - 已推送到 Harbor 的测试镜像和测试 tag
4. 真实部署链路联调
   - 用户视角：
     - 差异页可识别目标缺失服务
     - 用户可直接发起部署并查看健康检查结果
   - 进入前先准备 Kubernetes 测试集群、namespace、workload、Agent 到 K8s API 的访问能力
   - 同时准备最小联调样本：
     - 至少 1 套可重复部署的 K8s manifests
     - 1 套“已有服务更新 tag”验证样例
     - 1 套“缺失服务首次部署”验证样例
5. 远程项目环境部署与管理达标
   - 用户视角：
     - 用户可在平台中完成远程环境管理、远程发版、远程部署、结果追踪
   - 进入前先准备完整远程联调环境和环境负责人
   - 样本材料需同时齐全：
     - 测试镜像
     - Jenkins 流水线与构建脚本
     - K8s manifests
6. 审计、权限、持久化补强
   - 在主链路跑通后继续补强

## 当前进度结论

- 已完成并已推送：
  - 工程初始化
  - 前端页面与 mock 数据
  - 后端 mock API
  - 登录/权限 mock
  - 更新日志
  - 前后端 API 联调
  - 数据库与迁移基础
  - Redis Stream mock Agent worker
  - mock 集成 adapter
- 当前本地阶段：
  - 第 1 步“发布/部署详情闭环”已基本完成，处于待提交收口状态
- 默认下一步：
  - 第 2 步“Agent 协议闭环”

## 用户视角页面测试顺序

1. 登录页
2. 环境管理页 / Agent 管理页
3. 基线列表页 / 基线详情页
4. 差异对比页
5. 创建发布页
6. 发布列表页 / 发布详情页
7. 部署列表页 / 部署详情页
8. 用户 / 角色 / 权限页

## 页面统一验收问题

1. 用户能不能进入页面
2. 用户能不能看懂当前状态
3. 用户能不能完成核心动作
4. 动作后状态会不会更新
5. 出错时有没有明确提示和兜底展示

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

## 任务 4：登录与权限 mock

```text
请补充登录与基础权限能力。前端实现登录页、路由守卫、顶部栏用户信息、退出登录、用户管理、角色管理、环境权限配置页面。后端实现 mock 登录、当前用户、用户列表、角色列表、权限列表接口。先使用 mock token，不接真实 SSO，不保存真实密码。
```

重点：

- 未登录访问业务页面时跳转登录页
- 登录后能进入工作台
- 顶部栏展示当前用户和角色
- 退出登录后清理本地 token
- 用户、角色、环境权限页面使用 mock 数据
- 写操作入口按角色做基础按钮级控制

重点接口：

- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/me`
- `GET /api/users`
- `GET /api/roles`
- `GET /api/permissions`

## 任务 5：更新日志页面与 mock API

```text
请补充更新日志页面，用于记录平台每个小版本上线后的迭代与更新情况。前端在系统管理下增加更新日志菜单和页面，后端提供 mock changelog API。
```

重点：

- 页面展示版本号、上线时间、更新类型、新增功能、修复问题、已知问题、发布人
- 支持按版本号、更新类型、关键词筛选
- 数据先来自 `mocks/changelog.json` 或后端 mock repository
- 暂不做富文本编辑和审批发布

重点接口：

- `GET /api/changelog`

## 任务 6：前后端联调

```text
请把前端 mock JSON 替换为后端 API 调用。保留一个 mock 模式开关，便于没有后端时前端仍可运行。
```

## 任务 7：数据库模型

```text
请根据 docs/domain-model.md 设计 PostgreSQL 表结构和 GORM model，添加数据库迁移。先支持环境、Agent、基线、发布单、部署任务、用户、角色、权限、更新日志、操作日志。
```

## 任务 8：任务与 Agent 模拟

```text
请使用 Redis Stream 实现平台任务队列和 mock Agent worker。创建发布单或部署任务后，mock worker 按步骤更新任务状态并追加日志。
```

## 任务 9：测试和收口

```text
请根据 docs/acceptance-criteria.md 补充前端关键交互测试和后端 API 测试，确保 MVP 验收项通过。
```
