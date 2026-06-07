# 运维发布交付平台开发说明

本目录用于交给 VSCode + Codex 进行后续开发。开发时请以 `docs/PRD.md` 为业务依据，以 `design/ops-release-console-v3.html` 为视觉和页面结构参考，以 `docs/domain-model.md`、`docs/state-machine.md`、`docs/api-contract.md`、`mocks/` 为开发约束。

## 技术栈

- 前端：Vue 3 + Vite + TypeScript + Pinia + Vue Router + Element Plus
- 后端：Go 1.22+ + Gin + GORM
- 数据库：PostgreSQL 16
- 缓存与任务队列：Redis 7，MVP 使用 Redis Stream 承载 Agent 任务队列
- 实时日志：MVP 使用 HTTP 轮询，后续可升级 WebSocket/SSE
- 部署方式：docker-compose
- 第三方集成：MVP 先使用 mock adapter，保留 Jenkins、Harbor、Kubernetes、GitLab、ArgoCD、Nacos 的 adapter 接口

## MVP 开发顺序

1. 前端静态页面工程化，还原 HTML 原型。
2. 接入 mock JSON，完成页面路由、表格、筛选、抽屉、步骤和日志展示。
3. Go 后端提供 REST API，先返回 mock 数据。
4. 建立 PostgreSQL 表结构和基础 CRUD。
5. 接入 Redis Stream，实现 Agent 任务模型的模拟流转。
6. 后续逐步替换 mock adapter 为真实 Jenkins、Harbor、K8s 集成。

## 重要约束

- 不要把 HTML 原型中的示例数据直接当成业务规则。
- 所有状态枚举以 `docs/state-machine.md` 为准。
- 所有接口字段以 `docs/api-contract.md` 为准。
- V1 不做复杂审批流、灰度发布、完整离线交付和真实 CMDB。

## 本地启动

### 前端

```bash
cd frontend
npm install
npm run dev
```

默认访问：`http://localhost:5173`

### 后端

```bash
cd backend
go mod tidy
go run ./cmd/server
```

健康检查：`http://localhost:8080/healthz`

可选环境变量：

- `APP_PORT`：后端监听端口，默认 `8080`
- `DATABASE_DSN`：PostgreSQL 连接串，不配置时跳过数据库迁移
- `REDIS_ADDR`：Redis 地址，不配置时不启动 mock Agent worker
- `INTEGRATION_MODE`：第三方系统 adapter 模式，默认 `mock`；当前只支持 `mock`

### Docker Compose

```bash
docker compose up --build
```

服务端口：

- frontend: `http://localhost:5173`
- backend: `http://localhost:8080`
- postgres: `localhost:5432`
- redis: `localhost:6379`
