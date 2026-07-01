# API 合约草案

统一响应格式：

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "requestId": "req-20260607-0001"
}
```

分页响应：

```json
{
  "items": [],
  "page": 1,
  "pageSize": 20,
  "total": 100
}
```

## 环境

### GET /api/environments

查询环境列表。

Query：`type`、`status`、`keyword`、`page`、`pageSize`

Response data：

```json
{
  "items": [
    {
      "id": "env-project-x-prod",
      "name": "项目 X 生产",
      "code": "project-x-prod",
      "type": "PROJECT",
      "networkMode": "AGENT",
      "status": "HEALTHY",
      "agentStatus": "ONLINE",
      "lastCheckAt": "2026-06-07T12:40:00+08:00"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

### POST /api/environments/{id}/check

触发环境依赖检查。仅本地环境支持平台 adapter 直连检查；远程环境由 Agent 心跳、探测和任务回传维护状态，平台不对远程环境执行直连检查。

Response data：

```json
{
  "environmentId": "env-project-x-prod",
  "status": "HEALTHY",
  "checkedAt": "2026-06-07T13:20:00+08:00",
  "checks": [
    {
      "component": "kubernetes",
      "status": "HEALTHY",
      "message": "kubernetes connection is available",
      "checkedAt": "2026-06-07T13:20:00+08:00"
    },
    {
      "component": "harbor",
      "status": "HEALTHY",
      "message": "registry connection is available",
      "checkedAt": "2026-06-07T13:20:00+08:00"
    }
  ]
}
```

## 基础资源

K8s、Harbor、Jenkins 是独立平台资源，由产品引用和使用。当前后端仍复用 `Environment` 模型承载产品部署范围，用户侧按“产品管理”理解。

产品资源绑定分两类：

- `BUILD_SOURCE`：平台侧构建和版本来源。本地产品绑定本地 K8s namespace、本地 Harbor project、本地 Jenkins view；远程产品绑定本地 Harbor project、本地 Jenkins view。
- `RUNTIME_TARGET`：远程运行目标。远程 K8s namespace、远程 Harbor project 由项目环境 Agent 上报后，在平台产品配置中映射到产品；一个产品可映射多个 namespace 和多个 Harbor project。

Agent 配置只保存平台地址、注册 token、远程 K8s/Harbor 连接信息，不保存 namespace/project 与产品的对应关系。

资源接口原则：

- 新增/编辑请求使用用户视角字段，不要求用户填写 `credentialRef`。
- 响应不返回明文 kubeconfig、密码或 token，只返回非敏感元数据、状态、最近检查信息和缓存列表。
- 资源状态由系统测试连接或刷新探测维护，用户不能手工更新。
- 刷新探测成功时更新缓存；失败时保留旧缓存并记录失败原因。
- 基础资源管理中的 K8s、Harbor、Jenkins 由平台后端探测，用于本地产品资源范围和远程产品的本地构建/版本来源。
- 远程产品的远程 K8s/Harbor 状态由 Agent 后续上报；Jenkins 属于平台侧本地基础资源，Agent 不连接 Jenkins。

### POST /api/kubernetes-clusters

创建 K8s 资源。请求可以传 kubeconfig 内容或上传文件后的引用，后端保存凭据并生成内部凭据引用。

Request data：

```json
{
  "id": "k8s-local-test",
  "name": "本地测试 K3s",
  "apiServer": "https://k8s.example.com:6443",
  "kubeconfig": "<masked>",
  "context": "default"
}
```

Response data：

```json
{
  "id": "k8s-local-test",
  "name": "本地测试 K3s",
  "apiServer": "https://k8s.example.com:6443",
  "context": "default",
  "status": "UNKNOWN",
  "lastCheckAt": "",
  "probeMessage": "",
  "namespaces": []
}
```

### POST /api/harbor-registries

创建 Harbor 资源。必须支持 HTTP Harbor 和 HTTPS Harbor。

Request data：

```json
{
  "id": "harbor-local",
  "name": "本地 Harbor",
  "url": "http://registry.example.com:5000",
  "scheme": "http",
  "username": "admin",
  "password": "<masked>",
  "insecureSkipTLSVerify": false
}
```

Response data：

```json
{
  "id": "harbor-local",
  "name": "本地 Harbor",
  "url": "http://registry.example.com:5000",
  "scheme": "http",
  "username": "admin",
  "insecureSkipTLSVerify": false,
  "status": "UNKNOWN",
  "lastCheckAt": "",
  "probeMessage": "",
  "projects": []
}
```

### POST /api/jenkins-instances

创建 Jenkins 资源。

Request data：

```json
{
  "id": "jenkins-local",
  "name": "本地 Jenkins",
  "url": "http://jenkins.example.com:8080",
  "username": "root",
  "token": "<masked>",
  "insecureSkipTLSVerify": false
}
```

Response data：

```json
{
  "id": "jenkins-local",
  "name": "本地 Jenkins",
  "url": "http://jenkins.example.com:8080",
  "username": "root",
  "insecureSkipTLSVerify": false,
  "status": "UNKNOWN",
  "lastCheckAt": "",
  "probeMessage": "",
  "views": [],
  "jobs": []
}
```

### POST /api/{resourceType}/{id}/test

测试单个资源连接。`resourceType` 可为 `kubernetes-clusters`、`harbor-registries`、`jenkins-instances`。

Response data：

```json
{
  "id": "harbor-local",
  "name": "本地 Harbor",
  "url": "http://registry.example.com:5000",
  "scheme": "http",
  "username": "admin",
  "insecureSkipTLSVerify": false,
  "status": "HEALTHY",
  "lastCheckAt": "2026-06-07T13:20:00+08:00",
  "probeMessage": "connection is available",
  "projects": []
}
```

### POST /api/{resourceType}/{id}/refresh

刷新资源探测缓存。K8s 刷新 namespaces，Harbor 刷新 projects，Jenkins 刷新 views/jobs。远程资源应返回 Agent 任务信息或异步探测状态。

Response data：

```json
{
  "id": "k8s-project-x",
  "name": "项目 K8s",
  "apiServer": "https://k8s.example.com:6443",
  "context": "default",
  "status": "HEALTHY",
  "lastCheckAt": "2026-06-07T13:20:00+08:00",
  "probeMessage": "connection is available",
  "namespaces": ["default", "project-x-prod"]
}
```

## Agent

### GET /api/agents

查询 Agent 列表。

Response data：

```json
{
  "items": [
    {
      "id": "agent-project-x",
      "name": "agent-project-x",
      "environmentId": "env-project-x-prod",
      "environmentName": "项目 X 生产",
      "version": "1.3.2",
      "status": "ONLINE",
      "capabilities": ["remote-probe", "k8s-api", "http-check"],
      "lastHeartbeatAt": "2026-06-07T12:40:12+08:00",
      "currentTaskId": "REL-20260607-031"
    }
  ],
  "page": 1,
  "pageSize": 20,
  "total": 1
}
```

V1 页面依赖字段：

- `status`：判断 Agent 是否在线；离线 Agent 会阻断其绑定项目环境的远程发布/部署。
- `capabilities`：展示 Agent 是否具备远程资源上报、Kubernetes API 访问、健康检查等执行能力。
- `lastHeartbeatAt`：展示最近心跳，辅助判断真实联调前 Agent 是否可用。
- `currentTaskId`：展示最近或当前执行任务，辅助用户从 Agent 管理页跳转排查。

### POST /api/agents/register-token

生成 Agent 注册 token。

Request：

```json
{
  "agentId": "agent-project-x-prod",
  "ttlMinutes": 10
}
```

Response data：

```json
{
  "agentId": "agent-project-x-prod",
  "token": "agtr_1781750628",
  "expiresAt": "2026-06-07T13:20:00+08:00",
  "configText": "# 平台侧登记的 Agent 唯一标识。首次注册建议直接使用平台页面生成的值。\nAGENT_ID=agent-project-x-prod\n\n# 首次注册时建议留空；在平台页面认领 Agent 后，Agent 会通过心跳同步绑定关系。\nAGENT_ENVIRONMENT_ID=\n\n# Agent 可出站访问的平台后端 API 地址。部署到项目环境前，请改成该机器可访问的平台地址。\nPLATFORM_URL=http://platform.example.com:8080\n\n# 首次注册可留空；使用 -f 配置文件启动时，注册成功后 Agent 会自动写回平台签发的运行令牌。\nAGENT_TOKEN=\n\n# 一次性注册密钥，使用一次后失效。\nAGENT_REGISTER_TOKEN=agtr_1781750628\n\nAGENT_MODE=remote-probe\nAGENT_HEALTH_PORT=18080\nAGENT_POLL_INTERVAL_SECONDS=5\nAGENT_HEARTBEAT_INTERVAL_SECONDS=15\nAGENT_HTTP_TIMEOUT_SECONDS=10\nAGENT_MAX_TASKS=1\nAGENT_CAPABILITIES=remote-probe,k8s-api,http-check\n\n# 远程 Kubernetes 连接配置。Agent 通过 Kubernetes API 上报资源，namespace 与产品的对应关系在平台维护。\nAGENT_KUBECONFIG=\n\n# 远程 Harbor 连接配置。Agent 只负责上报 project、镜像和 tag，project 与产品的对应关系在平台维护。\nAGENT_HARBOR_URL=\nAGENT_HARBOR_USERNAME=\nAGENT_HARBOR_PASSWORD=\nAGENT_HARBOR_INSECURE_SKIP_TLS_VERIFY=false",
  "installCommand": "# same as configText, kept for old clients"
}
```

`configText` 是页面展示和复制的 Agent 配置文本；`installCommand` 仅为兼容旧前端保留，不应再放 shell 启动命令。

### POST /api/agents/{id}/heartbeat

Agent 上报心跳。V1 必须校验首次注册后签发的长期 Agent token，用于把 Agent 标记为在线，并刷新版本、能力和最近心跳时间。

Request：

```json
{
  "version": "1.3.3",
  "capabilities": ["remote-probe", "k8s-api", "http-check"]
}
```

### POST /api/agent-tasks/lease

Agent 主动向平台领取待执行发布/部署任务。V1 项目环境发版/部署采用 Agent 出站模型：Agent 独立部署在项目环境内或可访问项目环境的 Linux 主机上，平台创建发布/部署任务后只在平台侧登记任务；Agent 通过可访问的平台 API 主动领取任务、上报执行状态并回传结果。本地环境默认平台 adapter 直连，不走该 Agent 前置。

Request：

```json
{
  "agentId": "agent-project-x",
  "environmentId": "env-project-x-prod",
  "maxTasks": 1,
  "leaseSeconds": 300
}
```

Response data：

```json
{
  "leased": true,
  "leaseId": "lease-REL-20260607-031",
  "task": {
    "id": "REL-20260607-031",
    "type": "release",
    "action": "jenkins_agent_release",
    "agentId": "agent-project-x",
    "environmentId": "env-project-x-prod",
    "services": [
      {
        "name": "order-api",
        "namespace": "prod",
        "image": "harbor.local/project-x/order-api:20260609-001"
      }
    ],
    "callback": {
      "stepUrl": "https://platform.local/api/agent-tasks/REL-20260607-031/steps",
      "logUrl": "https://platform.local/api/agent-tasks/REL-20260607-031/logs",
      "resultUrl": "https://platform.local/api/agent-tasks/REL-20260607-031/result"
    }
  }
}
```

平台不得依赖访问项目环境 Agent endpoint，也不得向 Agent 主动推送任务。项目环境默认平台不可连通，Agent 只支持出站访问平台 API。V1 当前使用 `/api/agent-tasks/lease` 作为任务领取/租约接口；旧实验接口不作为项目环境发版/部署主链路。

## 基线

### POST /api/baselines

从环境采集运行态并生成基线。

Request：

```json
{
  "sourceEnvironmentId": "env-local-prod",
  "name": "local-prod-20260607-1530",
  "purpose": "项目 X 交付"
}
```

Response `data` 重点字段：

```json
{
  "id": "BL-20260609-0004",
  "sourceEnvironmentId": "env-local-prod",
  "sourceEnvironmentName": "本地生产环境",
  "serviceCount": 3,
  "status": "DRAFT",
  "snapshotSource": "本地生产环境/local-prod",
  "snapshotCollectedAt": "2026-06-09T14:30:00+08:00",
  "snapshotMode": "REAL_RUNTIME",
  "snapshotTaskId": "snapshot-bl-20260609-0004",
  "items": []
}
```

### GET /api/baselines

查询基线列表。

### GET /api/baselines/{id}

查询基线详情、运行态快照元数据和服务清单。

### POST /api/baselines/{id}/lock

锁定基线。

### POST /api/baselines/{id}/compare

对比来源基线和目标环境。

Request：

```json
{
  "targetEnvironmentId": "env-project-x-prod",
  "refreshTargetRuntime": true
}
```

## 发布

### GET /api/releases

查询发布单列表。V1 列表同时展示服务发版和服务部署任务，前端按 `type` 区分用户语义：

- `SERVICE_RELEASE`：目标环境已有服务的迭代发版，不返回 `sourceBaselineId`，通过 `releaseSource`、构建任务和镜像元数据说明来源。
- `SERVICE_DEPLOYMENT`：目标环境缺失服务的首次部署，必须返回 `sourceBaselineId`，用于说明来源基线/生产环境。

Response item：

```json
{
  "id": "REL-20260607-031",
  "type": "SERVICE_RELEASE",
  "releaseSource": "JENKINS_JOB",
  "executionMode": "JENKINS_AGENT",
  "targetEnvironmentName": "项目 X 生产",
  "agentName": "agent-project-x",
  "status": "JENKINS_QUEUED",
  "progress": 30,
  "agentTaskId": "REL-20260607-031",
  "buildId": "42",
  "buildStatus": "QUEUED",
  "buildUrl": "https://jenkins.local/job/user-service-release/1",
  "imageRepository": "harbor.local/project-x/user-service",
  "imageTag": "20260607-a1b2c3",
  "imageDigest": "sha256:9f2c0b6d4a1e7c8b5d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9876543210abcdef12",
  "createdAt": "2026-06-07T13:20:00+08:00"
}
```

### POST /api/releases

创建发布单。

服务发版用于目标环境中已经存在的服务，不基于来源基线创建。平台调用 Jenkins adapter 真实触发 Jenkins 构建，不提供 mock 构建开关。

服务发版支持两种来源：

- `JENKINS_JOB`：选择与产品 Jenkins view 匹配、且已绑定到服务的 Jenkins Pipeline，执行构建 jar/dist、制作镜像并推送到本地 Harbor。缺少产品 Jenkins 绑定、服务 Pipeline 绑定或 Jenkins 触发失败时，接口返回失败，不创建假发布单。
- `LOCAL_HARBOR_IMAGE`：扫描本地 Harbor 上该服务的镜像版本，选择镜像 tag 发版；该路径不需要选择或触发 Jenkins Job。

上述两种来源最终都需要通过项目环境中运行的 Agent 同步到项目环境，完成项目 Harbor 镜像同步和 workload tag 更新。本地环境默认由平台侧直连链路或现有 GitOps 链路处理，不需要 Agent。

Request：

```json
{
  "type": "SERVICE_RELEASE",
  "releaseSource": "JENKINS_JOB",
  "targetEnvironmentId": "env-project-x-prod",
  "agentId": "agent-project-x",
  "serviceIds": ["svc-user"],
  "jenkins": {
    "jobName": "project-x-user-service-release",
    "branch": "release/20260607",
    "parameters": {
      "SERVICE_NAME": "user-service",
      "TARGET_ENV": "project-x-prod"
    }
  }
}
```

Response data：

```json
{
  "id": "REL-20260607-031",
  "status": "JENKINS_QUEUED",
  "executionMode": "JENKINS_AGENT",
  "agentTaskId": "REL-20260607-031",
  "buildId": "42",
  "buildStatus": "QUEUED",
  "createdAt": "2026-06-07T13:20:00+08:00"
}
```

基于本地 Harbor 镜像 tag 发版：

```json
{
  "type": "SERVICE_RELEASE",
  "releaseSource": "LOCAL_HARBOR_IMAGE",
  "targetEnvironmentId": "env-project-x-prod",
  "agentId": "agent-project-x",
  "serviceIds": ["svc-user"],
  "image": {
    "repository": "harbor.local/project-x/user-service",
    "tag": "20260607-a1b2c3",
    "digest": "sha256:9f2c0b6d4a1e7c8b5d3e2f1a0b9c8d7e6f5a4b3c2d1e0f9876543210abcdef12"
  }
}
```

服务部署前，可先通过基线对比确认目标缺失服务范围：

```json
{
  "type": "SERVICE_DEPLOYMENT",
  "sourceBaselineId": "BL-20260607-0001",
  "targetEnvironmentId": "env-project-x-prod",
  "agentId": "agent-project-x",
  "serviceIds": ["svc-user", "svc-payment", "svc-web"],
  "options": {
    "autoRollback": true,
    "skipWorkloadError": true,
    "refreshTargetRuntime": true
  }
}
```

服务部署不直接走 Jenkins，前端应创建部署任务：

```text
POST /api/deploy-tasks
```

### GET /api/releases/{id}

查询发布详情、步骤、日志索引、失败定位建议。

Response data 重点字段：

```json
{
  "id": "REL-20260607-031",
  "type": "SERVICE_RELEASE",
  "releaseSource": "JENKINS_JOB",
  "targetEnvironmentName": "项目 X 生产",
  "status": "PARTIAL_FAILED",
  "agentTaskId": "REL-20260607-031",
  "steps": [
    {
      "name": "HTTP 健康检查",
      "status": "FAILED",
      "message": "web-console 返回 503，order-service 超时"
    }
  ],
  "failures": [
    {
      "serviceName": "web-console",
      "reason": "HTTP 503",
      "suggestion": "检查 Nginx upstream、服务端口、ConfigMap 与 Pod 日志"
    }
  ],
  "actionRecords": [
    {
      "action": "FAIL_FAST",
      "operator": "system",
      "status": "FAILED",
      "message": "order-service 健康检查超时，任务转入部分失败",
      "occurredAt": "2026-06-07T15:43:44+08:00"
    }
  ],
  "auditSummary": {
    "operator": "li.si",
    "targetEnvironmentName": "项目 X 生产",
    "affectedServices": ["user-service", "web-console", "order-service"],
    "result": "PARTIAL_FAILED",
    "failedStep": "HTTP 健康检查",
    "lastAction": "FAIL_FAST",
    "lastActionAt": "2026-06-07T15:43:44+08:00"
  },
  "logs": []
}
```

### POST /api/releases/{id}/retry

重试失败发布或失败服务。

### POST /api/releases/{id}/rollback

回滚到上一 tag。

## Agent 任务

V1 Agent 任务只记录真实发布/部署任务。发布/部署创建后写入平台任务租约队列，Agent 通过出站 API 领取任务并回传真实执行状态。

### GET /api/agent-tasks/{id}/status

查询 Agent 任务状态和日志。状态和日志来自真实 Agent 回传。

当任务队列或 Agent 回传不可用时，接口返回明确错误或空状态，前端展示“暂无 Agent 回传”。

Response data：

```json
{
  "enabled": true,
  "status": {
    "taskId": "REL-20260607-031",
    "type": "release",
    "action": "jenkins_agent_release",
    "status": "SUCCESS",
    "step": "finished",
    "agentId": "agent-project-x",
    "updatedAt": "2026-06-09T14:32:00+08:00"
  },
  "logs": [
    "remote harbor replication started",
    "workload image updated successfully"
  ]
}
```

### POST /api/agent-tasks/{id}/steps

Agent 回调平台，回传当前步骤状态。

Request：

```json
{
  "step": "sync-image",
  "status": "RUNNING"
}
```

### POST /api/agent-tasks/{id}/logs

Agent 回调平台，追加任务日志。

Request：

```json
{
  "line": "remote harbor replication started"
}
```

### POST /api/agent-tasks/{id}/result

Agent 回调平台，回传最终执行结果。`SUCCESS` 或 `FAILED` 会释放 Agent 当前任务占用。

Request：

```json
{
  "status": "SUCCESS",
  "message": "workload image updated successfully"
}
```

## 部署任务

### GET /api/deploy-tasks

查询部署任务列表。

V1 列表用于用户确认“目标缺失服务首次部署”的状态，不再以旧脚本/部署包为主线。列表项必须能直接展示来源基线、目标环境、缺失服务、Agent 任务、当前步骤和下一步动作。

Response data：

```json
{
  "items": [
    {
      "id": "DEP-20260607-009",
      "type": "SERVICE_DEPLOYMENT",
      "productName": "项目 X",
      "targetEnvironmentName": "项目 X 生产",
      "sourceBaselineId": "BL-20260607-0001",
      "source": "BL-20260607-0001",
      "missingServiceCount": 2,
      "serviceNames": ["order-web", "payment-worker"],
      "currentStep": "恢复 MinIO",
      "progress": 46,
      "status": "RUNNING",
      "agentName": "agent-project-x",
      "agentTaskId": "DEP-20260607-009",
      "nextAction": "等待人工确认数据恢复结果"
    }
  ],
  "page": 1,
  "pageSize": 10,
  "total": 1
}
```

### POST /api/deploy-tasks

创建部署任务。

服务部署时使用该接口。平台基于来源基线/生产环境和目标环境的差异结果，选择 `MISSING_IN_TARGET` 服务创建部署任务，由 Agent 或后续真实 adapter 在目标环境创建 workload、同步镜像并做健康检查。

Request：

```json
{
  "type": "SERVICE_DEPLOYMENT",
  "sourceBaselineId": "BL-20260607-0001",
  "targetEnvironmentId": "env-project-x-prod",
  "agentId": "agent-project-x",
  "serviceIds": ["svc-new-order"],
  "options": {
    "syncImage": true,
    "createWorkload": true,
    "healthCheck": true
  }
}
```

Response data：

```json
{
  "id": "DEP-20260607-004",
  "status": "PENDING",
  "executionMode": "AGENT",
  "agentTaskId": "DEP-20260607-004",
  "createdAt": "2026-06-07T13:20:00+08:00"
}
```

### GET /api/deploy-tasks/{id}

查询部署任务详情、步骤和日志。

详情页必须能支撑失败路径验收：当前步骤、日志、执行记录、重试、跳过、人工确认，以及 Agent 轮询状态。

Response data 重点字段：

```json
{
  "id": "DEP-20260607-009",
  "productName": "项目 X",
  "targetEnvironmentName": "项目 X 生产",
  "source": "BL-20260607-0001",
  "status": "RUNNING",
  "agentTaskId": "DEP-20260607-009",
  "steps": [
    {
      "id": "step-7",
      "order": 7,
      "name": "人工确认：数据恢复结果",
      "type": "MANUAL_CONFIRM",
      "status": "WAITING_CONFIRM"
    }
  ],
  "actionRecords": [
    {
      "action": "WAIT_CONFIRM",
      "operator": "system",
      "status": "PENDING_CONFIRM",
      "message": "等待人工确认数据恢复结果",
      "occurredAt": "2026-06-07T16:14:03+08:00"
    }
  ],
  "auditSummary": {
    "operator": "wang.wu",
    "targetEnvironmentName": "项目 X 生产",
    "affectedServices": ["order-web", "payment-worker"],
    "result": "RUNNING",
    "failedStep": "",
    "lastAction": "WAIT_CONFIRM",
    "lastActionAt": "2026-06-07T16:14:03+08:00"
  },
  "logs": []
}
```

### POST /api/deploy-tasks/{id}/steps/{stepId}/retry

重试步骤。

### POST /api/deploy-tasks/{id}/steps/{stepId}/skip

跳过步骤。

### POST /api/deploy-tasks/{id}/steps/{stepId}/confirm

人工确认步骤。

## 错误码

| code | 说明 |
|---|---|
| OK | 成功 |
| VALIDATION_ERROR | 参数错误 |
| NOT_FOUND | 资源不存在 |
| AGENT_OFFLINE | Agent 离线 |
| INTEGRATION_FAILED | 第三方系统调用失败 |
| TASK_CONFLICT | 任务状态冲突 |
| PERMISSION_DENIED | 无权限 |
