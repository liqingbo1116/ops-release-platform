# Ops Release Agent

V1 Agent 用于部署在远程项目环境中，平台侧默认无法直连该环境。Agent 只通过出站请求推送心跳、拉取任务租约、回传步骤/日志/结果，不要求平台主动访问远程项目环境。

## V1 运行模式

- 研发阶段必须优先使用二进制方式部署和调试 Agent，不使用 docker-compose 作为日常开发启动方式。
- 默认使用 `AGENT_MODE=remote-probe`，用于真实探测项目环境绑定的 K8s namespace 和 Harbor project。
- `AGENT_MODE=remote-probe` 不会模拟发布/部署成功；遇到发布/部署任务会返回失败，等后续真实执行器接入。
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
- `AGENT_ENVIRONMENT_ID`：Agent 已认领后的项目环境 ID。
  首次注册时建议留空，平台页面认领后 Agent 会通过心跳同步绑定关系；重启后也可以填写已认领的环境 ID，减少一次同步等待。
- `PLATFORM_URL`：Agent 可出站访问的平台 API 地址。
  必须包含协议和端口，例如 `http://10.0.0.12:8080` 或 `https://platform.example.com`。不要写成前端地址，也不要缺少协议头。
- `AGENT_TOKEN`：Agent 调用平台 API 时使用的令牌。
  首次注册时可为空，平台会通过 `AGENT_REGISTER_TOKEN` 换取运行令牌。Agent 注册成功后需要保存平台返回的运行令牌，后续重启优先使用 `AGENT_TOKEN`。
- `AGENT_REGISTER_TOKEN`：Agent 首次注册令牌。
  在平台 Agent 管理页面生成，写入项目环境机器上的 `agent.env`；该令牌用于建立 Agent 与平台连接，使用一次后失效。
- `AGENT_MODE`：Agent 执行模式。
  V1 研发阶段默认 `remote-probe`，只做真实远程资源探测。仅本地协议验证时才可临时使用 `mock`，不要用于真实环境验收。
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
  远程探测阶段建议配置 `remote-probe,kubectl,http-check`。
- `AGENT_KUBECONFIG`：Agent 机器上的 kubeconfig 文件路径。
  平台下发环境绑定的 namespace 后，Agent 会执行 `kubectl --kubeconfig <path> get namespace <namespace>` 验证是否存在。
- `AGENT_HARBOR_URL`、`AGENT_HARBOR_USERNAME`、`AGENT_HARBOR_PASSWORD`：项目环境 Harbor 访问配置。
  Agent 会调用 `/api/v2.0/projects/{project}` 验证环境绑定的 Harbor project。

推荐配置方式：

- 研发阶段远程调试：在平台 Agent 管理页面生成注册 Token，复制生成的启动指令到项目环境机器，按该项目环境实际情况填写 K8s、Harbor 配置。
- 正式部署前联调：确认 `PLATFORM_URL` 使用远程主机可访问的实际平台地址，确认 `AGENT_HEALTH_PORT` 不与宿主机冲突。
- 注册成功后：平台页面会先显示 `在线 / 待认领`，需要在页面上认领到对应项目环境；保存运行令牌到 `AGENT_TOKEN`，清理已使用的 `AGENT_REGISTER_TOKEN`，不要把真实 token 提交进 Git。

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

启动成功信号：

```bash
curl http://127.0.0.1:${AGENT_HEALTH_PORT:-18080}/healthz
```

平台侧应看到 Agent 在线。认领到项目环境后，在 Agent 管理页面点击“远程探测”，环境状态会进入验证中并在 Agent 回传结果后更新为 `HEALTHY`、`DEGRADED` 或 `UNHEALTHY`。

## Docker Compose 部署

docker-compose 只作为后续正式上线部署方式预留，V1 研发调试阶段不要依赖它。

启动后 Agent 会按配置周期向平台发送心跳，并轮询 `/api/agent-tasks/lease` 获取任务。平台侧发起远程探测后，Agent 会访问项目环境中的 Kubernetes、Harbor 并回传真实结果。Jenkins 属于平台侧本地基础资源，由平台后端直连，不由项目环境 Agent 访问。

## 环境准备边界

环境未准备好时也可以注册 Agent，但远程探测结果会如实反映不可用原因：

- 远程项目环境 Agent 心跳在线。
- 缺少 `AGENT_KUBECONFIG` 时，K8s namespace 检查返回 `DEGRADED`。
- Harbor 未配置地址或账号时，对应检查返回 `DEGRADED`。
- 资源不存在或无法访问时，对应检查返回 `UNHEALTHY`。
- 全部绑定资源可访问时，环境状态更新为 `HEALTHY`。

后续接入真实服务版本来源、发布和部署执行器时，会继续沿用当前 Agent 注册、认领、心跳、租约、回传协议。
