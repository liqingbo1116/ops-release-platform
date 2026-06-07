<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>创建发布单</h1>
        <p>强化批量服务选择、风险确认、自动回滚与跳过异常 workload 策略。</p>
      </div>
      <el-button type="primary">提交发布单</el-button>
    </div>

    <div class="two-col">
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>发布配置</strong><el-tag>基线差异发布</el-tag></div></template>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="发布类型"><el-select model-value="BASELINE_DIFF"><el-option label="基线差异发布" value="BASELINE_DIFF" /></el-select></el-form-item>
          <el-form-item label="目标环境"><el-select model-value="env-project-x-prod"><el-option label="项目 X 生产 / project-x-prod" value="env-project-x-prod" /></el-select></el-form-item>
          <el-form-item label="来源基线"><el-select model-value="BL-20260607-0001"><el-option label="BL-20260607-0001 / local-prod" value="BL-20260607-0001" /></el-select></el-form-item>
          <el-form-item label="执行 Agent"><el-select model-value="agent-project-x"><el-option label="agent-project-x / 在线" value="agent-project-x" /></el-select></el-form-item>
        </el-form>
      </el-card>

      <ReleaseRiskPanel v-model:options="options" :selected-count="selectedIds.length" />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <strong>已选择 <span class="mono">{{ selectedIds.length }}</span> 个服务</strong>
          <el-input v-model="keyword" placeholder="搜索服务、namespace、tag" clearable />
        </div>
        <div class="top-actions">
          <el-button @click="selectPublishable">选择全部可发布</el-button>
          <el-button @click="selectedIds = []">清空选择</el-button>
        </div>
      </div>
      <ServiceDiffTable v-model:selected-ids="selectedIds" :items="filteredItems" />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import ReleaseRiskPanel from '@/components/ReleaseRiskPanel.vue'
import ServiceDiffTable from '@/components/ServiceDiffTable.vue'
import { mockData } from '@/api/mockData'

const keyword = ref('')
const selectedIds = ref<string[]>(mockData.diffResult.items.filter((item) => item.publishable).map((item) => item.serviceId))
const options = ref({
  autoRollback: true,
  skipWorkloadError: true,
  refreshTargetRuntime: true,
  auditLog: true,
})

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return mockData.diffResult.items
  return mockData.diffResult.items.filter((item) =>
    `${item.serviceName} ${item.namespace} ${item.sourceTag} ${item.targetTag ?? ''}`.toLowerCase().includes(q),
  )
})

function selectPublishable() {
  selectedIds.value = filteredItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
}
</script>
