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

Request：

```json
{
  "type": "BASELINE_DIFF",
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

### GET /api/releases/{id}

查询发布详情、步骤、日志索引、失败定位建议。

### POST /api/releases/{id}/retry

重试失败发布或失败服务。

### POST /api/releases/{id}/rollback

回滚到上一 tag。

## 部署任务

### GET /api/deploy-tasks

查询部署任务列表。

### POST /api/deploy-tasks

创建部署任务。

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
