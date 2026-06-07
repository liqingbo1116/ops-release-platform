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

MVP 后端提供任务队列接口：

```text
Agent -> 平台：心跳
Agent -> 平台：拉取任务
Agent -> 平台：上报任务步骤状态
Agent -> 平台：上报日志片段
```

开发期可以实现 `mock-agent-worker`，模拟 Agent 执行镜像同步、kubectl、shell 和健康检查。
