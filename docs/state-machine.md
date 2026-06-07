# 状态流转

## Agent 状态

```text
ONLINE -> HEARTBEAT_TIMEOUT -> OFFLINE
OFFLINE -> ONLINE
HEARTBEAT_TIMEOUT -> ONLINE
```

| 状态 | 说明 |
|---|---|
| ONLINE | 最近心跳正常 |
| HEARTBEAT_TIMEOUT | 超过心跳阈值但未判定离线 |
| OFFLINE | 长时间无心跳或主动下线 |

## 基线状态

```text
DRAFT -> LOCKED -> DEPRECATED
DRAFT -> DEPRECATED
```

| 状态 | 说明 |
|---|---|
| DRAFT | 已生成但未锁定 |
| LOCKED | 可用于正式交付，不允许修改服务项 |
| DEPRECATED | 已废弃，仅保留历史记录 |

## 发布单状态

```text
PENDING_CONFIRM -> RUNNING -> SUCCESS
PENDING_CONFIRM -> CANCELLED
RUNNING -> PARTIAL_FAILED -> RUNNING
RUNNING -> FAILED
RUNNING -> CANCELLED
FAILED -> RUNNING
FAILED -> ROLLED_BACK
PARTIAL_FAILED -> SUCCESS
PARTIAL_FAILED -> FAILED
```

| 状态 | 说明 |
|---|---|
| PENDING_CONFIRM | 等待确认 |
| RUNNING | 执行中 |
| PARTIAL_FAILED | 部分服务失败，可重试 |
| FAILED | 发布失败 |
| SUCCESS | 发布成功 |
| ROLLED_BACK | 已回滚 |
| CANCELLED | 已取消 |

## 部署任务状态

```text
PENDING -> RUNNING -> WAITING_CONFIRM -> RUNNING -> SUCCESS
RUNNING -> FAILED -> RUNNING
RUNNING -> CANCELLED
WAITING_CONFIRM -> CANCELLED
```

| 状态 | 说明 |
|---|---|
| PENDING | 已创建未执行 |
| RUNNING | 执行中 |
| WAITING_CONFIRM | 等待人工确认 |
| FAILED | 失败，可重试或跳过 |
| SUCCESS | 成功 |
| CANCELLED | 取消 |

## 部署步骤状态

```text
PENDING -> RUNNING -> SUCCESS
RUNNING -> WAITING_CONFIRM -> SUCCESS
RUNNING -> FAILED -> RUNNING
FAILED -> SKIPPED
```

## 服务差异状态

| 状态 | 说明 | 默认动作 |
|---|---|---|
| CONSISTENT | 来源和目标一致 | 不发布 |
| NEED_UPDATE | tag/digest 不一致 | 同步镜像并更新 tag |
| MISSING_IN_TARGET | 目标缺少服务 | 需确认新增部署 |
| WORKLOAD_ERROR | workload 异常 | 不可发布 |
| NOT_PUBLISHABLE | 不满足发布条件 | 禁止勾选 |
