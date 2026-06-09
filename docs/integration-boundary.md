# 第三方系统集成边界

MVP 原则：先跑通平台业务闭环，所有外部系统先通过 adapter 层隔离。开发期默认使用 mock adapter，后续替换为真实 adapter。

| 系统 | V1 策略 | 说明 |
|---|---|---|
| Jenkins | Mock first | 保留触发 job、查询构建状态接口 |
| Harbor | Mock first | 保留查询镜像、同步镜像、校验 digest 接口 |
| Kubernetes | Mock first | 保留采集 workload、更新 image、rollout 状态接口 |
| GitLab | 暂不真实写入 | 本地 GitOps 场景后续接入 |
| ArgoCD | 暂不真实接入 | 本地环境后续可触发 sync |
| Nacos | 只展示状态 | V1 不替代配置中心 |
| MySQL | 部署步骤模拟 | 不做数据库结构自动 diff |
| MinIO | 部署步骤模拟 | 不做真实对象恢复 |
| OSS | 部署步骤模拟 | 后续支持部署包下载 |

## Adapter 接口建议

```go
type KubernetesAdapter interface {
    ListWorkloads(ctx context.Context, envID string) ([]Workload, error)
    SetImage(ctx context.Context, envID string, req SetImageRequest) error
    GetRolloutStatus(ctx context.Context, envID string, workload string) (RolloutStatus, error)
}

type RegistryAdapter interface {
    GetImage(ctx context.Context, image string, tag string) (ImageInfo, error)
    SyncImage(ctx context.Context, req SyncImageRequest) error
}

type JenkinsAdapter interface {
    TriggerBuild(ctx context.Context, req BuildRequest) (BuildResult, error)
    GetBuildStatus(ctx context.Context, buildID string) (BuildStatus, error)
}
```

## Agent 通信边界

环境分为本地环境与项目环境。本地环境按平台侧可连通处理，可以由平台后端通过 adapter 直接访问本地 Jenkins、Harbor、Kubernetes 或 mock adapter，不需要 Agent。项目环境按平台侧不可连通处理，必须通过 Agent 接入；平台不得依赖访问项目环境 Agent endpoint，也不得向 Agent 主动推送任务。

项目环境由 Agent 主动出站访问平台 API，平台只负责登记待执行任务、保存状态和展示结果。Agent 负责主动领取/租约获取任务，并主动上报心跳、服务列表、镜像版本、步骤状态、日志和最终结果。

MVP 后端提供任务队列接口：

```text
Agent -> 平台：心跳
Agent -> 平台：领取/租约获取任务
Agent -> 平台：上报任务步骤状态
Agent -> 平台：上报日志片段
Agent -> 平台：上报最终结果
```

开发期既保留平台侧 `mock-agent-worker`，也提供独立远程 Agent 的 mock executor。独立 Agent 在真实外部组件准备好之前只做模拟执行：通过出站请求领取任务，模拟镜像同步、kubectl、shell 和健康检查，再通过平台回调接口上报步骤、日志和最终结果。该模式用于先验证项目环境不可直连条件下的远程发版/部署链路。

V1 Agent 协议约束：

- Agent 以 `docker compose` 部署在项目环境侧 Linux 主机，不作为 Kubernetes 内 workload 前置。
- Agent 当前只支持 `AGENT_MODE=mock`，真实 Jenkins、Harbor、Kubernetes executor 在环境准备后再接入。
- Agent 当前只支持 `AGENT_MAX_TASKS=1`，平台同一时间只向同一 Agent 下发一个运行中租约任务。
- 平台会回收过期租约并允许任务重新租约，避免 Agent 进程退出或网络中断后任务永久停留在运行态。
- 远程 Agent mock 验证阶段只需要 Agent 主机、Docker/Compose 和到平台 API 的出站网络，不需要 Jenkins、Harbor、Kubernetes。
