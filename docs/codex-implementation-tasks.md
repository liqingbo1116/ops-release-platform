# Codex 实施任务拆分

将本项目交给 VSCode + Codex 开发时，建议按下面顺序逐步执行，不要一次性要求实现全部功能。

## 当前执行原则

- 先以实现 V1 功能闭环为主，不以优化为主。
- V1 最低目标不是“页面齐全”，而是“至少支持项目环境的部署与管理”。
- 当研发即将触达 Jenkins、Harbor、Kubernetes、真实 Agent 联调时，必须提前说明所需环境和准备项，由环境提供方先准备，再继续真实集成开发。
- 后续默认研发顺序以“V1 实现规划（当前执行基线）”为准：
  1. 远程 Agent 独立部署包
  2. Agent 主动领取/租约任务链路
  3. 远程 Agent mock executor
  4. 发布/部署详情与远程 Agent 回调收口
  5. 真实发布链路联调
  6. 真实部署链路联调
  7. 项目环境部署与管理达标
  8. 审计、权限、持久化补强
- 包体优化、构建 warning 清理、UI 打磨、纯重构清理都放在这条主链路之后，除非已经阻塞功能交付。

## 外部环境准备规则

- 允许先用 mock adapter 推进功能闭环，但不能默认真实环境“之后再说”。
- 只要某一步开始需要真实系统联调，就要提前列出需要的组件与最小准备条件。
- 环境分为本地环境与项目环境：本地环境默认平台可直连，不需要 Agent；项目环境默认平台不可直连，必须通过 Agent 接入。
- V1 默认 Agent 部署方案固定为：
  - Agent 运行在独立 Linux 主机
  - 使用 `docker compose` 部署
  - Agent 操作的目标环境是远程 Kubernetes 集群
  - 不把 Agent 自身先部署进 Kubernetes 作为 V1 前置条件
- V1 项目环境发版/部署的硬前提：
  - Agent 必须能独立部署到项目环境侧 Linux 主机
  - 项目环境默认平台不可连通，不能要求平台访问 Agent endpoint，也不能由平台向 Agent 主动推送任务
  - Agent 只支持出站访问平台 API，必须能主动领取/租约获取发布/部署任务和执行数据
  - Agent 必须能主动向平台上报心跳、服务列表、镜像版本、步骤状态、日志和最终结果
  - 以上三项未完成前，不能进入真实远程环境发版/部署验收
- 默认需要重点提前确认的外部组件：
  - Jenkins：测试 Job、凭证接入方式、可用视图或命名规则
  - Harbor/Registry：测试仓库、镜像推送与拉取权限、测试 tag 策略
  - Kubernetes：测试集群、namespace、服务账号或 kubeconfig 准备方式
  - Agent 运行节点：可部署位置、出站访问平台 API 的网络连通性、日志与结果回传路径
- 文档和 TODO 中应把这类准备要求写在对应研发阶段前，而不是放在最终联调时再补充。

## V1 实现规划（当前执行基线）

本节是 Codex 后续继续研发时的默认顺序。不要把运行态采集、基线、真实 Jenkins/Harbor/Kubernetes 集成排到远程 Agent 可部署和主动领取任务链路之前。

1. 远程 Agent 独立部署包
   - 实现 `agent/cmd/agent` 可运行进程
   - 实现配置读取、健康检查、任务领取客户端、结果回传客户端
   - 增加 `agent/Dockerfile`
   - 增加远程 Agent `docker-compose.yml`
   - 增加 `.env.example`，不包含真实密钥
   - 用户视角：
     - 环境负责人可在远程 Linux 主机用 `docker compose up -d` 启动 Agent
     - 平台能看到 Agent 在线或健康
   - 进入前先准备：
     - Linux 主机
     - `docker`
     - `docker compose`
     - Agent 到平台 API 的网络连通性
   - 不需要 Jenkins、Harbor、Kubernetes
2. Agent 主动领取任务链路
   - 创建发布/部署任务后，平台登记待执行任务 payload
   - Agent 主动调用平台任务领取/租约接口获取任务 payload
   - payload 包含任务类型、环境、服务、镜像、来源、步骤和回调信息
   - 支持幂等执行键、领取确认、租约超时、领取失败状态
   - Agent 执行后回传步骤状态、日志和最终结果
   - 用户视角：
     - 任务详情能看到“已被远程 Agent 领取”
     - 领取或租约失败时能看到明确失败提示
     - 日志来自远程 Agent 回传
   - 进入前先准备：
     - 第 1 步 Agent 可远程运行
     - Agent 可访问平台 API
3. 远程 Agent mock executor
   - 不接真实 Jenkins/Harbor/K8s
   - Agent 领取任务后模拟执行发布/部署步骤
   - 模拟 Jenkins 构建、Harbor 同步、K8s 部署/更新、健康检查
   - 回传步骤状态、日志、失败原因和最终结果
   - 用户视角：
     - 用户可以通过平台提交发布/部署任务
     - 远程 Agent 实际收到任务并模拟执行
     - 详情页展示远程 Agent 回传的过程和结果
   - 进入前只需要 Agent 主机和网络连通性
4. 发布/部署详情与远程 Agent 回调收口
   - 补 `agentTaskId` 关联、状态刷新、日志、失败原因
   - 用户视角：
     - 进入发布详情页可看到状态、步骤、日志、执行记录、发布报告
     - 进入部署详情页可看到状态、步骤、日志、执行记录
     - 失败或阻塞时可操作重试、跳过、人工确认、回滚
   - 若要真实远程验证，先完成第 1、2、3 步
5. 真实发布链路联调
   - 用户视角：
     - 创建发布时可选择 Jenkins Job 或 Harbor 镜像 tag
     - 用户不再手工查项目 Harbor 或手工改 tag
   - 进入前先完成第 1、2、3、4 步，并准备 Jenkins、Harbor/Registry、Agent 到两者的访问能力
   - 同时准备最小联调样本：
     - 测试服务源码仓库
     - `Dockerfile`
     - Jenkinsfile 或构建脚本
     - 已推送到 Harbor 的测试镜像和测试 tag
6. 真实部署链路联调
   - 用户视角：
     - 差异页可识别目标缺失服务
     - 用户可直接发起部署并查看健康检查结果
   - 进入前先完成第 1、2、3、4 步，并准备 Kubernetes 测试集群、namespace、workload、Agent 到 K8s API 的访问能力
   - 同时准备最小联调样本：
     - 至少 1 套可重复部署的 K8s manifests
     - 1 套“已有服务更新 tag”验证样例
     - 1 套“缺失服务首次部署”验证样例
7. 项目环境部署与管理达标
   - 用户视角：
     - 用户可在平台中完成远程环境管理、远程发版、远程部署、结果追踪
   - 进入前先准备完整远程联调环境和环境负责人
   - 样本材料需同时齐全：
     - 测试镜像
     - Jenkins 流水线与构建脚本
     - K8s manifests
8. 审计、权限、持久化补强
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
  - 平台侧发布/部署详情闭环已完成并推送
  - 平台侧 Agent 协议 mock-first 实现已完成并推送
  - 运行态快照与基线生成 mock 链路已完成并推送
  - 差异结果到服务发布/新增部署的端到端 mock 验证已完成并推送
  - 失败动作、审计影响范围、环境/Agent 准备状态 mock-first 验证已完成并推送
- 当前本地未提交阶段：
  - 已补齐独立 Agent 可运行进程：`agent/cmd/agent`
  - 已补齐 Agent 配置读取、健康检查、心跳上报、任务租约领取、回调上报客户端
  - 已补齐远程 Agent mock executor，先模拟 Jenkins、Harbor、K8s 执行步骤
  - 已补齐 `agent/Dockerfile`、`agent/docker-compose.yml`、`agent/.env.example`
  - 已补齐平台 `/api/agent-tasks/lease` 主动领取/租约接口
  - 已补齐发布/部署任务入队时的 `agentId`、`environmentId`、payload 绑定
  - 已补齐 Agent 租约领取后回调步骤、日志、最终结果的本地回归测试
- 当前缺口：
  - 独立 Agent 包尚未在真实远程 Linux 主机用 `docker compose` 验证
  - 尚未完成跨主机网络下的心跳、租约领取、mock 日志、最终结果回传验收
  - 尚未接入真实 Jenkins、Harbor/Registry、Kubernetes
  - 因此真实远程发版/部署测试仍需等 Agent 远程验证和外部组件准备完成后开始
- 默认下一步：
  - 先在远程 Linux 主机部署 `agent/docker-compose.yml`
  - 验证 Agent 只通过出站访问平台 API 完成心跳、任务领取、mock 执行、日志和结果回传
  - 再收口发布/部署详情页对远程 Agent 回调状态的展示
  - Jenkins、Harbor/Registry、Kubernetes 和测试样例准备完成后，再进入真实执行联调

## 已完成的平台侧 mock-first Agent 协议

当前代码已支持：

- Agent 心跳：刷新在线状态、版本、能力、心跳时间
- 平台侧 mock 任务状态：从内存协议存储读取发布/部署任务状态
- Agent 步骤回传：更新当前步骤和状态
- Agent 日志回传：追加任务日志
- Agent 结果回传：更新最终状态并释放 Agent 当前任务
- 任务状态查询：详情页可读取 Agent 回传状态和日志
- Agent 管理页：从后端 Agent 列表读取在线状态、心跳和当前任务

当前实现已具备本地可运行的独立 Agent、Dockerfile、远程 `docker compose` 模板和 Agent 主动领取/租约链路。下一步必须先把该 Agent 部署到远程 Linux 主机验证出站链路；真实 Jenkins、真实 Harbor、真实 Kubernetes 仍未接入。

## 已完成的运行态快照与基线生成 mock 链路

当前代码已支持：

- 从来源环境生成基线时同步生成 mock 运行态服务清单
- 基线详情返回快照来源、采集时间、采集模式、快照任务 ID
- 基线详情页展示运行态快照元数据，便于用户确认基线来自哪个环境和哪次采集
- 基线对比继续兼容 `NEED_UPDATE`、`MISSING_IN_TARGET`、`WORKLOAD_ERROR`、`CONSISTENT` 分类

真实 Kubernetes 运行态采集尚未接入。环境准备完成前，继续使用 `MOCK_RUNTIME` 模式验证页面和任务流。

## 已验证的差异到任务端到端 mock 链路

当前代码已支持并通过本地测试验证：

- 差异页选择 `NEED_UPDATE` 服务后进入创建发布页
- 差异页选择 `MISSING_IN_TARGET` 服务后进入创建部署页
- 创建发布/部署任务后跳转详情页并保留 `agentTaskId`
- 前端纯 mock 模式下也能读取 mock Agent 任务状态、当前步骤和日志
- 服务发版请求不依赖来源基线
- 服务部署请求继续携带来源基线，用于确认目标缺失服务范围

真实 Jenkins、Harbor、Kubernetes 未准备完成前，本步骤只验证用户路径和接口契约，不接真实外部组件。

## 已完成的失败动作与准备状态 mock-first 验证

当前代码已支持并通过本地测试验证：

- 发布重试会更新 mock Agent task status 为 `retry` / `RUNNING`
- 发布回滚会更新 mock Agent task status 为 `rollback` / `ROLLED_BACK`
- 部署步骤重试、跳过、人工确认会同步更新 mock Agent task status
- 发布/部署详情页展示操作者、目标环境、影响服务、结果、失败步骤和最近动作
- 环境页展示真实联调前的 Agent、Jenkins、Harbor/Registry、Kubernetes 准备项
- Agent 页展示 V1 默认 Linux + `docker compose` 部署假设，并提示离线 Agent 会阻断远程发布/部署
- 环境级权限失败在后端返回 `403 FORBIDDEN`，创建页会映射成用户可理解的权限提示

本地验证结果：

- 后端：`go test ./...`
- 前端单测：`npm run test:unit -- --run`，10 个测试文件、39 个用例通过
- 前端构建：`npm run build` 通过，仅保留第三方依赖 pure annotation warning

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
