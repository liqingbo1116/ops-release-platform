# 第三方系统集成边界

V1 原则：按阶段接入真实数据和真实外部系统。历史 mock adapter 只作为早期原型背景，不作为当前阶段完成标准。

| 系统 | V1 策略 | 说明 |
|---|---|---|
| Jenkins | 平台侧本地集成 | 本地 Jenkins 由平台后端直连，用于后续构建和版本来源；项目 Agent 不连接 Jenkins |
| Harbor | 平台侧本地 + 项目 Agent | 本地 Harbor 由平台后端直连；项目 Harbor 由 Agent 探测和后续执行 |
| Kubernetes | 本地直连 + 项目 Agent | 本地 K8s 由平台后端直连；项目 K8s 由 Agent 探测和后续执行 |
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

环境分为本地环境与项目环境。本地环境按平台侧可连通处理，可以由平台后端通过 adapter 直接访问本地 Jenkins、Harbor、Kubernetes，不需要 Agent。项目环境按平台侧不可连通处理，必须通过 Agent 接入项目 K8s/Harbor；平台不得依赖访问项目环境 Agent endpoint，也不得向 Agent 主动推送任务。Jenkins 属于平台侧本地基础资源，不属于项目 Agent 探测范围。

项目环境由 Agent 主动出站访问平台 API，平台只负责登记待执行任务、保存状态和展示结果。Agent 负责主动领取/租约获取任务，并主动上报心跳、服务列表、镜像版本、步骤状态、日志和最终结果。

MVP 后端提供任务队列接口：

```text
Agent -> 平台：心跳
Agent -> 平台：领取/租约获取任务
Agent -> 平台：上报任务步骤状态
Agent -> 平台：上报日志片段
Agent -> 平台：上报最终结果
```

历史 mock Agent 链路只作为早期本地验证背景，不再作为 V1 当前阶段完成标准。Agent 管理与远程探测阶段必须使用真实 Agent 进程、真实注册、真实心跳、真实 token 校验和真实远程探测。如项目侧 Agent、网络或外部资源未准备好，应记录阻塞并提示用户需要配合的部署或配置项，不能用 mock 数据替代完成。

V1 Agent 协议约束：

- 研发阶段 Agent 以直接构建出的二进制在 Linux 主机上启动，便于远程调试、改配置和快速重启。
- 正式上线部署阶段再验证 `docker compose` 部署方式，不作为当前研发阶段门禁。
- Agent 管理页面生成一次性注册密钥，并展示平台地址、密钥有效期和二进制启动配置示例。
- Agent 首次注册成功后由平台签发长期 Agent token；心跳、任务租约、远程探测、执行回传必须校验长期 Agent token。
- 平台支持 `在线 / 待认领` Agent。待认领 Agent 只能显示未归属探测摘要，不能进入正式产品/服务视图，也不能执行发布/部署任务。
- Agent 当前只支持 `AGENT_MAX_TASKS=1`，平台同一时间只向同一 Agent 下发一个运行中租约任务。
- 平台会回收过期租约并允许任务重新租约，避免 Agent 进程退出或网络中断后任务永久停留在运行态。
- 如需用户在项目环境部署 Agent、修改配置、开放网络或准备项目 K8s/Harbor 访问，必须及时说明具体配置、执行命令和成功信号。

## 运行态数据主从边界

V1 平台不作为 K8s workload、Harbor 镜像、Jenkins Pipeline 或 Git 配置的主数据系统。真实产品环境是服务存在性的唯一依据：

- 本地产品的服务存在性以平台直连本地 K8s/Harbor/Jenkins 的探测结果为准。
- 远程产品的服务存在性以 Agent 上报的远程 K8s/Harbor 运行态为准；Jenkins 和本地 Harbor 仍由平台侧本地集成提供。
- 平台保存的服务纳管关系只是发版/部署关注范围和配置关系，不代表平台拥有该服务。
- 远程产品由 Agent 主动上报运行态快照。平台收到 Agent 上报后必须主动处理纳管关系、资源变化和服务展示状态，不能等用户手工点击刷新后才修正。
- 本地产品可以由页面刷新或后台探测触发平台直连探测；平台一旦拿到最新探测结果，也必须立即处理纳管关系、资源变化和服务展示状态。
- 如果已纳管服务在真实产品环境中被删除或消失，平台在收到最新探测数据后必须自动解除纳管关系，主服务列表不再显示该服务，也不能继续发版。
- 如果未纳管服务在真实产品环境中被删除或消失，平台在收到最新探测数据后不再显示即可。
- 平台可以保留事件日志、资源变化记录和历史发布/部署记录，用于说明“服务消失、自动解除纳管、服务重新出现”等事实。
- V1 平台不提供真实删除操作；所有 UI 和 API 语义必须区分“解除纳管”和“删除真实服务”。
