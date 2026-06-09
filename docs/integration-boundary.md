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

开发期可以实现 `mock-agent-worker`，模拟 Agent 执行镜像同步、kubectl、shell 和健康检查。
