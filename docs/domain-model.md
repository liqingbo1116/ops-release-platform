# 领域模型

## Environment

环境是服务运行和交付的目标。

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| id | string | 是 | 环境 ID |
| name | string | 是 | 环境名称 |
| code | string | 是 | 环境编码 |
| type | enum | 是 | LOCAL / PROJECT / TEST / STAGING |
| networkMode | enum | 是 | DIRECT / AGENT / OFFLINE |
| clusterId | string | 否 | K8s 集群 ID |
| registryId | string | 否 | Harbor/镜像仓库 ID |
| agentId | string | 否 | 绑定 Agent |
| status | enum | 是 | HEALTHY / DEGRADED / OFFLINE / UNKNOWN |
| lastCheckAt | datetime | 否 | 最近连接测试时间 |

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
| productId | string | 是 | 产品 |
| targetEnvironmentId | string | 是 | 目标环境 |
| sourceType | enum | 是 | BASELINE / PACKAGE / SCRIPT |
| sourceRef | string | 是 | 基线 ID、部署包或脚本引用 |
| status | enum | 是 | 部署任务状态 |
| currentStepId | string | 否 | 当前步骤 |
| progress | int | 是 | 0-100 |
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
