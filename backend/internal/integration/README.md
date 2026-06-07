# Integration Adapters

MVP 阶段第三方系统统一通过 adapter 层隔离，默认只启用 mock adapter，不实现真实 Jenkins、Harbor、Kubernetes、GitLab、ArgoCD、Nacos 集成。

当前已预留：

- `JenkinsAdapter`：触发构建、查询构建状态
- `RegistryAdapter`：检查镜像仓库连接、查询镜像、同步镜像
- `KubernetesAdapter`：检查集群连接、采集 workload、更新 image、查询 rollout 状态

配置：

- `INTEGRATION_MODE=mock`：默认值，使用内置 mock adapter
- 其他模式当前会启动失败，避免误以为已经接入真实系统

后续真实接入时，在本目录内增加 real adapter 实现，并保持上层 service/API 调用接口稳定。
