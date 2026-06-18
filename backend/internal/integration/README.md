# Integration Adapters

第三方系统统一通过 adapter 层隔离。V1 环境管理阶段必须使用真实环境数据完成连通性校验，不能用 mock 数据替代主线研发结果。

当前已预留：

- `JenkinsAdapter`：触发构建、查询构建状态
- `RegistryAdapter`：检查镜像仓库连接、查询镜像、同步镜像
- `KubernetesAdapter`：检查集群连接、采集 workload、更新 image、查询 rollout 状态

配置：

- `INTEGRATION_MODE=mock`：仅允许研发早期或单元测试使用，环境管理主线不得用它验收。
- `INTEGRATION_MODE=real`：启用真实 Harbor/Kubernetes 连通性校验。缺少真实配置时后端启动失败。
- `LOCAL_HARBOR_URL`
- `LOCAL_HARBOR_USERNAME`
- `LOCAL_HARBOR_PASSWORD`
- `LOCAL_K8S_KUBECONFIG`
- `REMOTE_HARBOR_URL`
- `REMOTE_HARBOR_USERNAME`
- `REMOTE_HARBOR_PASSWORD`
- `REMOTE_K8S_KUBECONFIG`
- `INTEGRATION_HTTP_TIMEOUT_MS`

敏感值只能写入 `.secrets/integration-connections.env` 或对应 shell/PowerShell 本地脚本，不得写入代码、文档或提交到 git。环境记录通过 `clusterId`、`registryId` 选择逻辑配置，当前固定使用 `local` 和 `remote` 两组 ID；为空时按环境类型默认选择：`LOCAL` 使用 `local`，`PROJECT` 使用 `remote`。

当前 `real` 模式已接入 Harbor `/api/v2.0/systeminfo` 和 Kubernetes `/readyz` 连通性检查。Jenkins、镜像同步、工作负载发布仍未完成真实实现，进入对应 V1 阶段前必须先准备真实 Jenkins/Harbor/Kubernetes 信息并移除该阶段 mock 路径。
