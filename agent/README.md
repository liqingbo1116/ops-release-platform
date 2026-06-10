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

- `AGENT_ID`：平台侧登记的 Agent 唯一标识。
  建议按 `agent-<project>-<env>` 命名，例如 `agent-project-x-prod`。同一个平台内必须唯一，不能和其他环境共用。
- `AGENT_ENVIRONMENT_ID`：Agent 所属项目环境 ID。
  该值必须与平台里的环境记录一致，例如 `env-project-x-prod`。任务租约会按这个字段匹配环境。
- `PLATFORM_URL`：Agent 可出站访问的平台 API 地址。
  必须包含协议和端口，例如 `http://10.0.0.12:8080` 或 `https://platform.example.com`。不要写成前端地址，也不要缺少协议头。
- `AGENT_TOKEN`：Agent 调用平台 API 时使用的令牌。
  当前 mock 阶段可为空；后续平台接入真实鉴权后，这里应配置平台签发的专用 token。
- `AGENT_MODE`：Agent 执行模式。
  当前只能配置为 `mock`，用于在 Jenkins、Harbor、K8s 未接入前模拟执行任务。
- `AGENT_HEALTH_PORT`：Agent 本地健康检查端口。
  默认 `18080`。用于 `curl http://127.0.0.1:<port>/healthz` 或宿主机探活，不要求平台主动访问。
- `AGENT_POLL_INTERVAL_SECONDS`：任务租约轮询间隔，单位秒。
  默认 `5`。值越小，平台新任务被领取越快，但对平台 API 请求更频繁。
- `AGENT_HEARTBEAT_INTERVAL_SECONDS`：心跳上报间隔，单位秒。
  默认 `15`。值越小，平台在线状态更新越快，但心跳请求会更多。
- `AGENT_HTTP_TIMEOUT_SECONDS`：调用平台 API 的超时时间，单位秒。
  默认 `10`。建议覆盖常见网络抖动，但不要设得过长，否则故障恢复会变慢。
- `AGENT_MAX_TASKS`：单个 Agent 可并发执行的任务数。
  V1 当前必须为 `1`，不能改成其他值。
- `AGENT_CAPABILITIES`：Agent 能力声明，逗号分隔。
  当前主要用于平台展示和协议占位。`mock-executor` 应保留；`image-sync`、`kubectl`、`http-check` 表示该 Agent 未来会承载的执行能力范围。

推荐配置方式：

- 研发阶段远程调试：复制 `.env.example` 为 `agent.env`，只改 `AGENT_ID`、`AGENT_ENVIRONMENT_ID`、`PLATFORM_URL`，其余保持默认。
- 正式部署前联调：确认 `PLATFORM_URL` 使用远程主机可访问的实际平台地址，确认 `AGENT_HEALTH_PORT` 不与宿主机冲突。
- 正式鉴权接入后：补充 `AGENT_TOKEN`，不要把真实 token 提交进 Git。

## 研发阶段直接构建

本地直接构建 Agent 二进制：

```bash
cd agent
go build -o bin/ops-release-agent ./cmd/agent
```

远程机器准备配置并启动：

```bash
scp bin/ops-release-agent remote:/opt/ops-release-agent/
scp .env.example remote:/opt/ops-release-agent/agent.env
ssh remote 'cd /opt/ops-release-agent && ./ops-release-agent -f ./agent.env'
```

`-f` 用于显式指定配置文件路径，适合 systemd、ssh 或手工远程启动。配置文件格式与 `.env.example` 一致。

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
