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

Query：`type`、`networkMode`、`status`、`keyword`、`page`、`pageSize`

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

触发连接测试。项目环境由 Agent 执行。

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
      "message": "mock kubernetes connection is available",
      "checkedAt": "2026-06-07T13:20:00+08:00"
    },
    {
      "component": "harbor",
      "status": "HEALTHY",
      "message": "mock registry connection is available",
      "checkedAt": "2026-06-07T13:20:00+08:00"
    }
  ]
}
```

## Agent

### GET /api/agents

查询 Agent 列表。

### POST /api/agents/register-token

生成 Agent 注册 token。

Request：

```json
{
  "environmentId": "env-project-x-prod",
  "ttlMinutes": 10
}
```

Response data：

```json
{
  "token": "agt_7f92c1b8_20260607",
  "expiresAt": "2026-06-07T13:20:00+08:00",
  "installCommand": "curl -fsSL https://platform.local/agent/install.sh | bash -s -- --token agt_7f92c1b8_20260607 --server https://platform.local"
}
```

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

### GET /api/baselines

查询基线列表。

### GET /api/baselines/{id}

查询基线详情和服务清单。

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

### POST /api/releases

创建发布单。

服务发版用于目标环境中已经存在的服务，不基于来源基线创建。平台调用 Jenkins adapter，MVP 阶段使用 mock Jenkins adapter，不直接接真实 Jenkins。

服务发版支持两种来源：

- `JENKINS_JOB`：选择与 Jenkins 视图或特征 job 关联后的 Jenkins Job，执行构建 jar/dist、制作镜像并推送到本地 Harbor。
- `LOCAL_HARBOR_IMAGE`：扫描本地 Harbor 上该服务的镜像版本，选择镜像 tag 发版；该路径不需要选择或触发 Jenkins Job。

上述两种来源最终都需要通过项目环境中运行的 Agent 同步到项目环境，完成项目 Harbor 镜像同步和 workload tag 更新。本地环境现阶段仍维持 GitOps。

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
  "id": "REL-20260607-MOCK",
  "status": "JENKINS_QUEUED",
  "executionMode": "JENKINS_AGENT",
  "agentTaskId": "REL-20260607-MOCK",
  "buildId": "BUILD-MOCK-20260607",
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
    "digest": "sha256:mock-20260607-a1b2c3"
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

### POST /api/releases/{id}/retry

重试失败发布或失败服务。

### POST /api/releases/{id}/rollback

回滚到上一 tag。

## Agent 任务

### GET /api/agent-tasks/{id}/status

查询 Redis Stream mock Agent worker 写入的任务状态和日志。

当 Redis 未配置时，接口返回 `enabled=false`，前端应降级展示发布/部署详情中的静态日志。

Response data：

```json
{
  "enabled": true,
  "status": {
    "taskId": "REL-20260607-MOCK",
    "type": "release",
    "step": "finish",
    "status": "SUCCESS",
    "updatedAt": "2026-06-07T13:20:00+08:00"
  },
  "logs": [
    "[2026-06-07T13:19:58+08:00] RUNNING receive-task",
    "[2026-06-07T13:20:00+08:00] SUCCESS finish"
  ]
}
```

## 部署任务

### GET /api/deploy-tasks

查询部署任务列表。

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
  "id": "DEP-20260607-MOCK",
  "status": "PENDING",
  "createdAt": "2026-06-07T13:20:00+08:00"
}
```

### GET /api/deploy-tasks/{id}

查询部署任务详情、步骤和日志。

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
