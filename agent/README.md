# Ops Release Agent

V1 Agent 用于部署在远程项目环境中，平台侧默认无法直连该环境。Agent 只通过出站请求推送心跳、拉取任务租约、回传步骤/日志/结果，不要求平台主动访问远程项目环境。

## V1 运行模式

- 研发阶段必须优先使用二进制方式部署和调试 Agent，不使用 docker-compose 作为日常开发启动方式。
- 默认使用 `AGENT_MODE=remote-probe`，用于真实读取远程项目环境的 K8s namespace、workload、Harbor project 和镜像 tag。
- `AGENT_MODE=remote-probe` 不会模拟发布/部署成功；遇到发布/部署任务会返回失败，等后续真实执行器接入。
- 当前只支持 `AGENT_MAX_TASKS=1`，同一 Agent 同一时间只执行一个租约任务。
- 本地环境属于平台直连场景，不需要部署 Agent；项目环境必须通过 Agent 推送数据。

## 配置

复制示例配置为运行配置文件：

```bash
cp .env.example agent.conf
```

关键变量：

- `AGENT_ID`：平台侧登记的 Agent 唯一标识。
  建议按 `agent-<project>-<env>` 命名，例如 `agent-project-x-prod`。同一个平台内必须唯一，不能和其他环境共用。
- `AGENT_ENVIRONMENT_ID`：Agent 已认领后的项目环境 ID。
  首次注册时建议留空，平台页面认领后 Agent 会通过心跳同步绑定关系；重启后也可以填写已认领的环境 ID，减少一次同步等待。
- `PLATFORM_URL`：Agent 可出站访问的平台 API 地址。
  必须包含协议和端口，例如 `http://10.0.0.12:8080` 或 `https://platform.example.com`。不要写成前端地址，也不要缺少协议头。
- `AGENT_TOKEN`：Agent 调用平台 API 时使用的令牌。
  首次注册时可为空，平台会通过 `AGENT_REGISTER_TOKEN` 换取运行令牌。使用 `-f` 配置文件启动时，Agent 注册成功后会自动把运行令牌写回 `AGENT_TOKEN`，并清空已使用的一次性注册密钥。
- `AGENT_REGISTER_TOKEN`：Agent 首次注册令牌。
  在平台 Agent 管理页面生成，写入项目环境机器上的 `agent.conf`；该令牌用于建立 Agent 与平台连接，使用一次后失效。
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
  远程资源上报阶段建议配置 `remote-probe,k8s-api,http-check`。
- `AGENT_KUBECONFIG`：Agent 机器上的 kubeconfig 文件路径。
  Agent 会通过该配置直接访问 Kubernetes API 读取远程 K8s 资源并上报平台，不要求安装 `kubectl`；namespace 与产品的对应关系在平台产品配置中维护，不在 Agent 配置中指定。
- `AGENT_HARBOR_URL`、`AGENT_HARBOR_USERNAME`、`AGENT_HARBOR_PASSWORD`：项目环境 Harbor 访问配置。
  Agent 会读取远程 Harbor project、镜像和 tag 并上报平台；Harbor project 与产品的对应关系在平台产品配置中维护，不在 Agent 配置中指定。
  Agent 会优先通过 Harbor API 获取实际 registry 地址；如果 Harbor API 未返回，则平台会结合服务镜像 registry 与已选择的 Harbor project 推断候选值，并在服务纳管时由用户确认。

推荐配置方式：

- 研发阶段远程调试：在平台 Agent 管理页面生成注册密钥，复制生成的配置文本到项目环境机器，按该项目环境实际情况填写 K8s、Harbor 配置。
- 首次启动最少需要确认 `PLATFORM_URL` 和 `AGENT_REGISTER_TOKEN`。如果页面生成的 `PLATFORM_URL` 是 `127.0.0.1` 或 `localhost`，部署到项目环境机器前必须改成该机器可访问的平台后端 API 地址。
- 需要读取远程 K8s 运行资源时填写 `AGENT_KUBECONFIG`；需要读取远程 Harbor project、镜像和 tag 时填写 `AGENT_HARBOR_URL`、`AGENT_HARBOR_USERNAME`、`AGENT_HARBOR_PASSWORD`。
- 正式部署前联调：确认 `PLATFORM_URL` 使用远程主机可访问的实际平台地址，确认 `AGENT_HEALTH_PORT` 不与宿主机冲突。
- 注册成功后：平台页面会先显示 `在线 / 待认领`，需要在页面上认领到对应项目环境；Agent 会自动写回 `AGENT_TOKEN` 并清空已使用的 `AGENT_REGISTER_TOKEN`，不要把真实 token 提交进 Git。

## 研发阶段直接构建

本地直接构建 Agent 二进制：

```bash
cd agent
go build -o bin/ops-release-agent ./cmd/agent
```

远程机器准备配置并启动：

```bash
scp bin/ops-release-agent remote:/opt/ops-release-agent/
scp .env.example remote:/opt/ops-release-agent/agent.conf
ssh remote 'cd /opt/ops-release-agent && ./ops-release-agent -f ./agent.conf'
```

`-f` 用于显式指定配置文件路径，适合 systemd、ssh 或手工远程启动。配置文件格式与 `.env.example` 一致。

启动成功信号：

```bash
curl http://127.0.0.1:${AGENT_HEALTH_PORT:-18080}/healthz
```

平台侧应看到 Agent 在线。认领到项目环境后，平台通过刷新看到 Agent 上报的远程 K8s、Harbor 与服务镜像版本数据；后续在产品配置中选择哪些远程 namespace/project 属于该产品。

## Docker Compose 部署

docker-compose 只作为后续正式上线部署方式预留，V1 研发调试阶段不要依赖它。

启动后 Agent 会按配置周期向平台发送心跳，并轮询 `/api/agent-tasks/lease` 获取任务。平台刷新后可看到 Agent 上报的项目环境 Kubernetes、Harbor 和服务镜像版本数据。Jenkins 属于平台侧本地基础资源，由平台后端直连，不由项目环境 Agent 访问。

## 环境准备边界

环境未准备好时也可以注册 Agent，但远程资源上报会如实反映不可用原因：

- 远程项目环境 Agent 心跳在线。
- 缺少 `AGENT_KUBECONFIG` 时，无法上报远程 K8s namespace、workload 和运行镜像。
- Harbor 未配置地址或账号时，无法上报远程 Harbor project、镜像和 tag。
- Harbor 访问地址与推送 registry 不一致时，不需要在 Agent 配置中手工指定 registry；平台会优先使用 Harbor API 返回值，无法自动确认时会在服务纳管页面要求用户确认候选 registry。
- Agent 能上报资源，不代表资源已经属于某个产品；产品映射必须在平台侧维护。
- 一个产品可以映射多个远程 namespace 和多个远程 Harbor project。

后续接入真实服务版本来源、发布和部署执行器时，会继续沿用当前 Agent 注册、认领、心跳、租约、回传协议。
