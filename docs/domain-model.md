# 领域模型

## Environment

环境是服务运行和交付的目标。

V1 用户视角会把当前环境能力承接为产品部署范围：用户看到的主层级是“项目 -> 产品 -> 服务 -> 发布 / 部署”，不再把环境作为产品下面的额外层级暴露。已完成的环境资源绑定、状态、Agent 就绪能力继续复用到产品管理中。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 环境 ID |
| name | string | 是 | 环境名称 |
| code | string | 是 | 环境标识；创建时系统按环境名称自动生成，页面可见且可修改，保存后系统生成 `env-<code>` 作为环境 ID。远程 Agent 配置使用生成后的环境 ID |
| type | enum | 是 | LOCAL / PROJECT；页面展示为本地环境 / 远程环境 |
| deployTargetType | enum | 是 | KUBERNETES / DOCKER_COMPOSE；V1 当前实现 Kubernetes，docker-compose 只预留模型入口 |
| networkMode | enum | 是 | 内部字段：本地环境固定 DIRECT，远程环境固定 AGENT，页面不让用户选择 |
| clusterId | string | 否 | 默认 K8s 绑定的兼容字段；本地环境使用，远程环境为空 |
| namespace | string | 否 | 默认 K8s namespace 兼容字段；本地环境使用，远程 K8s 由 Agent 上报 |
| registryId | string | 否 | 默认 Harbor 绑定的兼容字段；本地/远程环境都可关联平台维护的本地 Harbor |
| registryProject | string | 否 | 默认 Harbor project 兼容字段；远程环境用于本地镜像来源和同步任务 |
| jenkinsId | string | 否 | 默认 Jenkins 绑定的兼容字段；本地/远程环境都可关联平台维护的 Jenkins |
| jenkinsView | string | 否 | 默认 Jenkins view 兼容字段；远程环境用于本地构建流水线范围 |
| bindings | EnvironmentResourceBinding[] | 否 | 环境与基础资源的作用域绑定模型，可包含多个 K8s namespace、Harbor project、Jenkins view |
| agentId | string | 否 | 绑定 Agent |
| status | enum | 是 | HEALTHY / DEGRADED / OFFLINE / UNKNOWN |
| lastCheckAt | datetime | 否 | 最近连接测试时间 |

V1 环境管理必须把 K8s、Harbor、Jenkins 作为平台可维护的资源主数据，不把它们隐藏在 `.secrets/` 中作为正式数据来源。同一个 K8s 集群、Harbor 仓库或 Jenkins 实例可以被多个本地环境复用，同一环境也可以绑定多个作用域。多作用域绑定的业务目的不是“为了绑定而绑定”，而是为后续环境内服务关联做准备：服务需要在某个环境内明确归属到实际使用的 K8s namespace、Harbor project；本地构建/版本来源还可以使用 Jenkins view/job。项目环境不由平台直连项目 K8s/Harbor，也不绑定 Jenkins view；项目 K8s/Harbor 由 Agent 探测和执行，Jenkins 由平台后端直连本地基础资源。

进入项目/产品模型后，K8s、Harbor、Jenkins 仍是全局基础资源，不直接归属项目。产品引用实际使用的资源范围，项目只通过绑定的产品间接获得资源使用视图。

## EnvironmentResourceBinding

环境资源绑定是环境和基础资源之间的作用域关系。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 绑定 ID |
| environmentId | string | 是 | 环境 ID |
| resourceType | enum | 是 | K8S / HARBOR / JENKINS |
| resourceId | string | 是 | 基础资源 ID |
| scopeType | enum | 是 | NAMESPACE / PROJECT / VIEW |
| scopeValue | string | 是 | namespace、Harbor project 或 Jenkins view |
| isDefault | bool | 是 | 是否为该资源类型的默认绑定；旧兼容字段从默认绑定回填 |

V1 环境管理阶段必须落地该绑定模型的数据库表与后端接口。前端可以先只维护每类资源的默认绑定，但接口必须能返回完整绑定列表。后续服务模型需要能引用或选择环境内的具体绑定范围，用于表达“这个服务在该环境使用哪个 namespace、哪个 Harbor project、哪个 Jenkins view/job”。发布单和部署任务创建时必须保存当次实际选用的 namespace、Harbor project、Jenkins view/job 快照，不能只依赖环境当前默认绑定。

`.secrets/` 只用于研发阶段本地启动前后端、Agent 或连接私有工具时装载敏感值。正式使用时，平台数据库保存资源主数据，凭证字段只保存 `credentialRef`，由后续正式凭证后端或部署环境提供真实密钥。

环境状态来自系统探测结果，用户不能手工维护。环境只选择已缓存或手工输入的作用域，不直接保存 kubeconfig、Harbor 密码或 Jenkins Token。

## Project

项目是最高层级的业务归属边界，例如项目A、项目B。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 项目 ID |
| name | string | 是 | 项目名称 |
| code | string | 是 | 项目标识 |
| description | string | 否 | 项目说明 |
| status | enum | 是 | ACTIVE / DISABLED |
| createdAt | datetime | 是 | 创建时间 |
| updatedAt | datetime | 是 | 更新时间 |

一个项目可以绑定一个或多个产品。项目是后续产品、服务、发布单、部署单、基线和权限的上层归属入口。

## Product

产品是项目下的服务集合和部署范围，例如数据中台、物联中台。V1 中产品复用当前环境模型承接资源范围和 Agent 就绪状态。基础资源不复制到产品内，产品只保存对全局基础资源及其作用域的引用。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 产品 ID；V1 可由环境 ID 迁移或映射 |
| projectId | string | 否 | 所属项目；未绑定时为空 |
| name | string | 是 | 产品名称 |
| code | string | 是 | 产品标识 |
| bindingStatus | enum | 是 | UNBOUND / BOUND / MOVING / DISABLED |
| deploymentScopeId | string | 是 | 产品部署范围；V1 对应当前环境记录 |
| agentId | string | 否 | 绑定 Agent |
| createdAt | datetime | 是 | 创建时间 |
| updatedAt | datetime | 是 | 更新时间 |

产品绑定规则：

- 一个项目可以绑定多个产品。
- 一个产品在 V1 中最多归属一个项目。
- `UNBOUND` 表示产品尚未绑定项目，可被项目认领。
- `BOUND` 表示产品已绑定项目，可进入服务、发布、部署流程。
- `MOVING` 表示产品正在更换项目，禁止创建新的发布/部署任务。
- `DISABLED` 表示产品停用，不进入新的发布/部署选择。
- 解绑或更换项目不能修改历史发布、部署和基线记录；历史记录必须保存当时的项目、产品、服务快照。

## Resource Probe Rules

K8s、Harbor、Jenkins 是独立资源，不是环境的内嵌字段。资源新增和编辑必须按用户视角设计表单：

- K8s：名称、kubeconfig 上传或粘贴、可选 context。
- Harbor：名称、地址、HTTP/HTTPS、用户名、密码、可选跳过 TLS 校验。
- Jenkins：名称、地址、用户名、密码或 API Token、可选跳过 TLS 校验。

`credentialRef` 是平台内部字段，由后端在保存凭据后生成或关联，前端表单不让用户填写。资源状态统一由测试连接或刷新探测更新，状态建议包含 `UNKNOWN`、`HEALTHY`、`UNHEALTHY`、`UNAUTHORIZED`、`UNREACHABLE`、`TLS_ERROR`，并保存 `lastCheckAt`、`probeMessage`。

资源探测结果需要缓存，供环境关联时快速选择：

- K8s 缓存 namespaces。
- Harbor 缓存 projects。
- Jenkins 缓存 views/jobs。

页面必须提供“刷新/重新探测”能力。刷新失败时保留旧缓存，更新失败原因，不把旧缓存清空。平台基础资源由平台后端 adapter 探测，并用于本地环境关联；远程环境资源不在平台基础资源表中维护，由 Agent 后续上报状态和运行数据。

## KubernetesCluster

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | K8s 集群资源 ID |
| name | string | 是 | 集群名称 |
| apiServer | string | 否 | Kubernetes API Server 地址；创建时 `apiServer` 与 `kubeconfig` 至少提供一个 |
| context | string | 否 | kubeconfig 中选择的 context |
| credentialRef | string | 否 | 内部凭据引用，不由用户填写，不保存明文 kubeconfig |
| kubeconfig | string | 否 | 仅请求字段，响应不返回明文 kubeconfig |
| status | enum | 是 | UNKNOWN / HEALTHY / UNHEALTHY / UNAUTHORIZED / UNREACHABLE / TLS_ERROR |
| lastCheckAt | datetime | 否 | 最近连接测试时间 |
| probeMessage | string | 否 | 最近连接测试结果或失败原因 |
| namespaces | string[] | 否 | 最近一次成功探测到的 namespace 列表 |

## HarborRegistry

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | Harbor/镜像仓库资源 ID |
| name | string | 是 | 仓库名称 |
| url | string | 是 | Harbor/镜像仓库地址，需支持 HTTP 和 HTTPS |
| scheme | enum | 是 | http / https |
| username | string | 否 | Harbor 用户名 |
| password | string | 否 | 仅请求字段，响应不返回明文密码 |
| insecureSkipTLSVerify | bool | 否 | HTTPS 自签或测试环境是否跳过 TLS 校验 |
| credentialRef | string | 否 | 内部凭据引用，不由用户填写，不保存明文账号密码 |
| status | enum | 是 | UNKNOWN / HEALTHY / UNHEALTHY / UNAUTHORIZED / UNREACHABLE / TLS_ERROR |
| lastCheckAt | datetime | 否 | 最近连接测试时间 |
| probeMessage | string | 否 | 最近连接测试结果或失败原因 |
| projects | string[] | 否 | 最近一次成功探测到的 Harbor project 列表 |

## JenkinsInstance

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | Jenkins 实例资源 ID |
| name | string | 是 | Jenkins 名称 |
| url | string | 是 | Jenkins 地址 |
| username | string | 否 | Jenkins 用户名 |
| token | string | 否 | 仅请求字段，响应不返回明文密码或 API Token |
| insecureSkipTLSVerify | bool | 否 | HTTPS 自签或测试环境是否跳过 TLS 校验 |
| credentialRef | string | 否 | 内部凭据引用，不由用户填写，不保存明文账号密码或 token |
| status | enum | 是 | UNKNOWN / HEALTHY / UNHEALTHY / UNAUTHORIZED / UNREACHABLE / TLS_ERROR |
| lastCheckAt | datetime | 否 | 最近连接测试时间 |
| probeMessage | string | 否 | 最近连接测试结果或失败原因 |
| views | string[] | 否 | 最近一次成功探测到的 Jenkins view 列表 |
| jobs | string[] | 否 | 最近一次成功探测到的 Jenkins job 列表 |

## Agent

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | Agent ID |
| name | string | 是 | Agent 名称 |
| environmentId | string | 是 | 绑定环境 |
| version | string | 是 | Agent 版本 |
| status | enum | 是 | ONLINE / HEARTBEAT_TIMEOUT / OFFLINE |
| capabilities | string[] | 是 | image-sync / kubectl / shell / http-check |
| lastHeartbeatAt | datetime | 否 | 最近心跳 |
| currentTaskId | string | 否 | 当前任务 |

## Service

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 服务 ID |
| productId | string | 是 | 所属产品 |
| name | string | 是 | 服务名 |
| namespace | string | 是 | K8s namespace |
| workloadName | string | 是 | Deployment/StatefulSet 名称 |
| workloadType | enum | 是 | DEPLOYMENT / STATEFUL_SET |
| imageRepository | string | 是 | 镜像仓库 |
| healthCheckPath | string | 否 | 健康检查路径 |

## EnvironmentBaseline

环境基线是某个环境在某个时间点的运行态快照。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 基线 ID |
| name | string | 是 | 基线名称 |
| sourceEnvironmentId | string | 是 | 来源环境 |
| productId | string | 否 | 产品分组 |
| serviceCount | int | 是 | 服务数量 |
| status | enum | 是 | DRAFT / LOCKED / DEPRECATED |
| purpose | string | 否 | 用途 |
| createdBy | string | 是 | 创建人 |
| createdAt | datetime | 是 | 创建时间 |
| lockedAt | datetime | 否 | 锁定时间 |

## BaselineServiceItem

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| baselineId | string | 是 | 基线 ID |
| serviceId | string | 是 | 服务 ID |
| serviceName | string | 是 | 服务名 |
| namespace | string | 是 | namespace |
| workloadName | string | 是 | workload |
| workloadType | enum | 是 | DEPLOYMENT / STATEFUL_SET |
| image | string | 是 | 完整镜像 |
| tag | string | 是 | 镜像 tag |
| digest | string | 否 | 镜像 digest |
| replicas | int | 否 | 期望副本数 |
| readyReplicas | int | 否 | 就绪副本数 |
| healthStatus | enum | 是 | HEALTHY / UNHEALTHY / UNKNOWN |

## ReleaseOrder

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 发布单 ID |
| type | enum | 是 | SINGLE_SERVICE / MULTI_SERVICE / BASELINE_DIFF / ROLLBACK |
| sourceBaselineId | string | 否 | 来源基线 |
| targetEnvironmentId | string | 是 | 目标环境 |
| agentId | string | 否 | 执行 Agent |
| status | enum | 是 | 发布单状态 |
| selectedServiceCount | int | 是 | 选择服务数 |
| createdBy | string | 是 | 创建人 |
| createdAt | datetime | 是 | 创建时间 |

## DeployTask

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 部署任务 ID |
| type | enum | 是 | V1 固定为 SERVICE_DEPLOYMENT |
| productId | string | 是 | 产品 |
| targetEnvironmentId | string | 是 | 目标环境 |
| sourceBaselineId | string | 是 | 来源基线 ID |
| sourceType | enum | 是 | V1 使用 BASELINE |
| sourceRef | string | 是 | 来源基线 ID |
| missingServiceCount | int | 是 | 目标环境缺失服务数 |
| serviceNames | string[] | 是 | 本次首次部署的服务名 |
| status | enum | 是 | 部署任务状态 |
| currentStepId | string | 否 | 当前步骤 |
| progress | int | 是 | 0-100 |
| agentName | string | 是 | 执行 Agent |
| agentTaskId | string | 是 | Agent 任务 ID |
| nextAction | string | 否 | 给用户的下一步处理提示 |
| createdBy | string | 是 | 创建人 |
| createdAt | datetime | 是 | 创建时间 |

## DeployStep

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 步骤 ID |
| deployTaskId | string | 是 | 部署任务 ID |
| name | string | 是 | 步骤名称 |
| type | enum | 是 | SHELL / KUBECTL / SQL / HTTP_CHECK / MANUAL_CONFIRM / STANDARD |
| status | enum | 是 | 步骤状态 |
| order | int | 是 | 执行顺序 |
| retryCount | int | 是 | 重试次数 |
| startedAt | datetime | 否 | 开始时间 |
| finishedAt | datetime | 否 | 结束时间 |
