<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境差异对比</h1>
        <p>文本搜索与状态筛选使用统一过滤逻辑，筛选条件组合生效。</p>
      </div>
      <div class="top-actions">
        <el-button>重新采集目标环境</el-button>
        <el-button type="primary" @click="$router.push('/releases/create')">按差异创建发布单</el-button>
      </div>
    </div>

    <div class="compare-head">
      <el-card shadow="never"><strong>来源基线</strong><div class="mono">{{ data.sourceBaselineId }}</div><p>本地生产 / 213 服务 / 已锁定</p></el-card>
      <div class="arrow-box">同步到</div>
      <el-card shadow="never"><strong>目标环境</strong><div class="mono">{{ data.targetEnvironmentId }}</div><p>Agent 在线 / 最近采集 12 秒前</p></el-card>
    </div>

    <div class="metric-grid">
      <MetricCard label="一致" :value="data.summary.consistent" tone="good" />
      <MetricCard label="需更新" :value="data.summary.needUpdate" tone="warn" />
      <MetricCard label="目标缺失" :value="data.summary.missingInTarget" tone="bad" />
      <MetricCard label="workload 异常" :value="data.summary.workloadError" tone="bad" />
      <MetricCard label="可发布服务" :value="selectedIds.length" foot="实际勾选数量" />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left wrap">
          <el-radio-group v-model="statusFilter">
            <el-radio-button label="ALL">全部</el-radio-button>
            <el-radio-button label="NEED_UPDATE">只看需更新</el-radio-button>
            <el-radio-button label="MISSING_IN_TARGET">只看目标缺失</el-radio-button>
            <el-radio-button label="WORKLOAD_ERROR">只看 workload 异常</el-radio-button>
            <el-radio-button label="NOT_PUBLISHABLE">只看不可发布</el-radio-button>
          </el-radio-group>
          <el-input v-model="keyword" placeholder="搜索服务、tag、namespace" clearable />
        </div>
        <div class="top-actions">
          <el-button @click="selectPublishable">批量选择可发布服务</el-button>
          <el-button type="primary">确认 {{ selectedIds.length }} 个服务</el-button>
        </div>
      </div>
      <ServiceDiffTable v-model:selected-ids="selectedIds" :items="filteredItems" />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MetricCard from '@/components/MetricCard.vue'
import ServiceDiffTable from '@/components/ServiceDiffTable.vue'
import { mockData } from '@/api/mockData'

const data = mockData.diffResult
const keyword = ref('')
const statusFilter = ref('ALL')
const selectedIds = ref<string[]>([])

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return data.items.filter((item) => {
    const statusMatched =
      statusFilter.value === 'ALL' ||
      item.diffStatus === statusFilter.value ||
      (statusFilter.value === 'NOT_PUBLISHABLE' && !item.publishable)
    const keywordMatched =
      !q || `${item.serviceName} ${item.namespace} ${item.sourceTag} ${item.targetTag ?? ''}`.toLowerCase().includes(q)
    return statusMatched && keywordMatched
  })
})

function selectPublishable() {
  selectedIds.value = filteredItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
}
</script>
