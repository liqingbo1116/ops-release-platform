# Integration Adapters

第三方系统统一通过 adapter 层隔离。V1 运行态必须使用真实环境数据完成连通性校验与发版触发，禁止使用 mock 模式或 mock 数据替代主线研发结果。

当前已预留：

- `JenkinsAdapter`：触发构建、查询构建状态
- `RegistryAdapter`：检查镜像仓库连接、查询镜像、同步镜像
- `KubernetesAdapter`：检查集群连接、采集 workload、更新 image、查询 rollout 状态

可选研发期连接配置：

- `LOCAL_HARBOR_URL`
- `LOCAL_HARBOR_USERNAME`
- `LOCAL_HARBOR_PASSWORD`
- `LOCAL_K8S_KUBECONFIG`
- `REMOTE_HARBOR_URL`
- `REMOTE_HARBOR_USERNAME`
- `REMOTE_HARBOR_PASSWORD`
- `REMOTE_K8S_KUBECONFIG`
- `INTEGRATION_HTTP_TIMEOUT_MS`

K8s 集群、Harbor 仓库、Jenkins 实例是平台可单独维护的资源主数据，不以 `.secrets/` 中的固定 `local` / `remote` 逻辑 ID 作为正式数据来源。资源记录保存非敏感连接信息和 `credentialRef`：

- K8s 集群：名称、API Server、`credentialRef`
- Harbor 仓库：名称、URL、`credentialRef`
- Jenkins 实例：名称、URL、`credentialRef`

环境记录只关联这些资源，并保存环境级作用域：

- K8s：`clusterId` + `namespace`
- Harbor：`registryId` + `registryProject`
- Jenkins：`jenkinsId` + `jenkinsView`

同一个 K8s 集群、Harbor 仓库或 Jenkins 实例可以被多个环境复用，不同环境通过 namespace、Harbor project、Jenkins view 隔离。

敏感值只能写入 `.secrets/integration-connections.env` 或对应 shell/PowerShell 本地脚本，不得写入代码、文档或提交到 git。`.secrets/` 只用于研发阶段本地启动和真实连通性测试；正式发布后必须由平台数据库中的资源主数据、`credentialRef` 和正式凭证后端或部署环境提供连接能力。

当前 Jenkins 发版触发与构建状态查询已接入真实 Jenkins API。服务发版选择 `JENKINS_JOB` 时，平台必须解析到产品绑定的 Jenkins 实例、Jenkins view 和服务绑定的 pipeline，并真实调用 Jenkins；缺少绑定、pipeline 不存在或 Jenkins 返回错误时，接口直接失败，不创建假发布单。

当前已接入 Harbor `/api/v2.0/systeminfo` 和 Kubernetes `/readyz` 连通性检查。镜像同步、工作负载发布进入对应 V1 阶段前必须先准备真实 Harbor/Kubernetes/Agent 信息，不允许通过模拟执行绕过真实链路。
