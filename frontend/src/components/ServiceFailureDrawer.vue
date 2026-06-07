<template>
  <el-drawer v-model="visible" :title="failure ? `${failure.serviceName} / 失败详情` : '失败详情'" size="420px">
    <div v-if="failure" class="drawer-stack">
      <div class="kv"><span>失败原因</span><strong>{{ failure.reason }}</strong></div>
      <div class="kv block"><span>建议处理动作</span><p>{{ failure.suggestion }}</p></div>
      <el-button type="primary">打开日志定位</el-button>
      <el-button>重试该服务</el-button>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
type Failure = {
  serviceName: string
  reason: string
  suggestion: string
}

const visible = defineModel<boolean>('visible', { required: true })
defineProps<{
  failure: Failure | null
}>()
</script>
