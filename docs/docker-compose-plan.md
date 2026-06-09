# docker-compose 规划

MVP docker-compose 目标服务：

```text
frontend: Vue/Vite 前端
backend: Go/Gin API
postgres: PostgreSQL 16
redis: Redis 7
```

建议端口：

| 服务 | 端口 |
|---|---|
| frontend | 5173 |
| backend | 8080 |
| postgres | 5432 |
| redis | 6379 |

建议环境变量：

```text
APP_ENV=local
DATABASE_URL=postgres://ops:ops@postgres:5432/ops_release?sslmode=disable
REDIS_ADDR=redis:6379
CREDENTIAL_MASTER_KEY=local-dev-only-change-me
MOCK_INTEGRATIONS=true
```

开发阶段可以先只提交 compose 文件和空服务，等前后端工程生成后再补 Dockerfile。

## 远程 Agent docker-compose

V1 项目环境 Agent 不跟随平台主 `docker-compose.yml` 部署。Agent 独立运行在项目环境侧或可访问项目环境的 Linux 主机上，使用 `agent/docker-compose.yml` 启动。

最小准备条件：

- Linux 主机
- `docker`
- `docker compose`
- Agent 主机可以出站访问平台 API
- 平台不需要访问 Agent 端口

当前 Agent 部署文件：

- `agent/Dockerfile`
- `agent/docker-compose.yml`
- `agent/.env.example`
- `agent/README.md`

真实 Jenkins、Harbor/Registry、Kubernetes 准备好之前，Agent 使用 mock executor 验证心跳、任务领取、步骤日志和最终结果回传。

V1 约束：

- `AGENT_MODE=mock`
- `AGENT_MAX_TASKS=1`
- 同一 Agent 同一时间只执行一个租约任务
- 租约过期后平台可重新下发任务，避免 Agent 重启或网络中断后任务永久卡住
- `agent/docker-compose.yml` 包含 `/healthz` healthcheck，可先验证容器健康再验证平台侧心跳
