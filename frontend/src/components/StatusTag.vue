<template>
  <el-tag :type="tag.type" effect="light" round>
    {{ tag.label }}
  </el-tag>
</template>

<script setup lang="ts">
const props = defineProps<{
  status: string
}>()

const statusMap: Record<string, { label: string; type: 'success' | 'warning' | 'danger' | 'info' | 'primary' }> = {
  HEALTHY: { label: '健康', type: 'success' },
  DEGRADED: { label: '需验证', type: 'warning' },
  UNKNOWN: { label: '未知', type: 'info' },
  ONLINE: { label: '在线', type: 'success' },
  OFFLINE: { label: '离线', type: 'danger' },
  UNBOUND: { label: '未绑定', type: 'warning' },
  NOT_REQUIRED: { label: '不需要', type: 'info' },
  LOCKED: { label: '已锁定', type: 'success' },
  DRAFT: { label: '草稿', type: 'warning' },
  PENDING: { label: '待执行', type: 'info' },
  PENDING_CONFIRM: { label: '待确认', type: 'warning' },
  RUNNING: { label: '执行中', type: 'primary' },
  WAITING_CONFIRM: { label: '等待确认', type: 'warning' },
  SUCCESS: { label: '成功', type: 'success' },
  PARTIAL_FAILED: { label: '部分失败', type: 'warning' },
  FAILED: { label: '失败', type: 'danger' },
  CANCELLED: { label: '已取消', type: 'info' },
  CONSISTENT: { label: '一致', type: 'success' },
  NEED_UPDATE: { label: '需更新', type: 'warning' },
  MISSING_IN_TARGET: { label: '目标缺失', type: 'danger' },
  WORKLOAD_ERROR: { label: 'workload 异常', type: 'danger' },
  NOT_PUBLISHABLE: { label: '不可发布', type: 'danger' },
}

const tag = statusMap[props.status] ?? { label: props.status, type: 'info' as const }
</script>
