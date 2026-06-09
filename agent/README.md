# Ops Release Agent

V1 Agent 用于部署在远程项目环境中，平台侧默认无法直连该环境。Agent 只通过出站请求推送心跳、拉取任务租约、回传步骤/日志/结果，不要求平台主动访问远程项目环境。

## V1 运行模式

- 当前只支持 `AGENT_MODE=mock`，用于在 Jenkins、Harbor、K8s 环境准备好前验证远程发版/部署闭环。
- 当前只支持 `AGENT_MAX_TASKS=1`，同一 Agent 同一时间只执行一个租约任务。
- 本地环境属于平台直连场景，不需要部署 Agent；项目环境必须通过 Agent 推送数据。

## 配置

复制示例配置：

```bash
cp .env.example .env
```

关键变量：

- `AGENT_ID`：平台侧登记的 Agent ID，例如 `agent-project-x`。
- `AGENT_ENVIRONMENT_ID`：Agent 所属项目环境 ID，例如 `env-project-x-prod`。
- `PLATFORM_URL`：Agent 可出站访问的平台 API 地址。
- `AGENT_TOKEN`：预留鉴权令牌，当前 mock 阶段可为空。
- `AGENT_MODE`：当前必须为 `mock`。
- `AGENT_MAX_TASKS`：当前必须为 `1`。

## Docker Compose 部署

```bash
docker compose config
docker compose up -d --build
docker compose ps
```

健康检查：

```bash
curl http://127.0.0.1:${AGENT_HEALTH_PORT:-18080}/healthz
```

启动后 Agent 会按配置周期向平台发送心跳，并轮询 `/api/agent-tasks/lease` 获取任务。平台侧创建服务发版或部署任务后，Agent 会在 mock executor 中回传步骤、日志和最终结果。

## 环境准备边界

环境未准备好前无需 Jenkins、Harbor、K8s。完成 docker-compose 部署后，可先验证：

- 远程项目环境 Agent 心跳在线。
- 平台创建服务发版任务后，Agent 可租约并回传结果。
- 平台创建服务部署任务后，Agent 可租约并回传结果。
- Agent 重启或网络中断导致租约过期后，平台可重新下发该任务。

接入真实 Jenkins、Harbor、K8s 后，再将 `mock` executor 替换为真实构建、镜像同步和 K8s 发布执行器。
