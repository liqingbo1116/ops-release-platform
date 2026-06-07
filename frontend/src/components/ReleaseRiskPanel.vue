<template>
  <el-card shadow="never">
    <template #header>
      <div class="panel-head">
        <strong>风险与策略确认</strong>
        <el-tag type="warning">提交前必读</el-tag>
      </div>
    </template>
    <div class="risk">
      <strong>发布前风险提示</strong>
      <p>
        当前已选择 {{ selectedCount }} 个服务。workload 异常和不可发布服务已禁用，提交时按实际勾选数量计算镜像同步与 K8s workload 更新数量。
      </p>
      <el-checkbox v-model="options.autoRollback">失败时自动回滚到上一 tag</el-checkbox>
      <el-checkbox v-model="options.skipWorkloadError">跳过 workload 异常服务</el-checkbox>
      <el-checkbox v-model="options.refreshTargetRuntime">发布前重新采集目标环境运行态</el-checkbox>
      <el-checkbox v-model="options.auditLog">记录发布审计与 Agent 日志</el-checkbox>
    </div>
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  selectedCount: number
}>()

const options = defineModel<{
  autoRollback: boolean
  skipWorkloadError: boolean
  refreshTargetRuntime: boolean
  auditLog: boolean
}>('options', { required: true })
</script>
