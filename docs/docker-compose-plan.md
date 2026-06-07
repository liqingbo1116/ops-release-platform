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
