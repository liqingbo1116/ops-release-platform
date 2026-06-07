# 技术栈与工程约束

## 前端

- Vue 3
- Vite
- TypeScript
- Pinia
- Vue Router
- Element Plus
- Axios
- ECharts，可选，用于后续趋势图

前端目录建议：

```text
frontend/
  src/
    api/
    assets/
    components/
    layouts/
    pages/
    router/
    stores/
    types/
    utils/
```

## 后端

- Go 1.22+
- Gin：HTTP API
- GORM：数据库访问
- PostgreSQL driver
- Redis client：Agent 任务队列、任务状态缓存
- Zap / slog：结构化日志

后端目录建议：

```text
backend/
  cmd/server/
  internal/api/
  internal/app/
  internal/domain/
  internal/repository/
  internal/service/
  internal/integration/
  internal/agent/
  internal/config/
  internal/middleware/
```

## 数据库

- PostgreSQL 16
- MVP 使用关系模型保存环境、Agent、基线、发布单、部署任务、日志索引
- 大日志正文可先存数据库 text，后续迁移到对象存储或日志系统

## 缓存与任务

- Redis 7
- MVP 使用 Redis Stream 模拟 Agent 任务下发与执行回传
- Agent 真接入前，后端提供 mock agent worker

## 部署

- docker-compose
- 服务包含：frontend、backend、postgres、redis
- MVP 不要求 Kubernetes 部署

## 第三方系统策略

MVP 默认使用 mock adapter：

- JenkinsAdapter
- HarborAdapter
- KubernetesAdapter
- GitLabAdapter
- ArgoCDAdapter
- NacosAdapter

真实接入时替换 adapter 实现，不改变上层业务服务接口。
