<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境差异对比</h1>
        <p>文本搜索与状态筛选使用统一过滤逻辑，筛选条件组合生效。</p>
      </div>
      <div class="top-actions">
        <el-button>重新采集目标环境</el-button>
        <el-button @click="goCreateByMode('SERVICE_DEPLOYMENT')">按缺失服务创建部署任务</el-button>
        <el-button type="primary" @click="goCreateByMode('SERVICE_RELEASE')">按需更新服务创建发布单</el-button>
      </div>
    </div>

    <div class="compare-head">
      <el-card shadow="never"><strong>来源基线</strong><div class="mono">{{ data.sourceBaselineId }}</div><p>{{ baselineSummary }}</p></el-card>
      <div class="arrow-box">同步到</div>
      <el-card shadow="never"><strong>目标环境</strong><div class="mono">{{ data.targetEnvironmentId }}</div><p>{{ targetSummary }}</p></el-card>
    </div>

    <div class="metric-grid">
      <MetricCard label="一致" :value="data.summary.consistent" tone="good" />
      <MetricCard label="需更新" :value="data.summary.needUpdate" tone="warn" />
      <MetricCard label="服务部署" :value="data.summary.missingInTarget" tone="bad" />
      <MetricCard label="workload 异常" :value="data.summary.workloadError" tone="bad" />
      <MetricCard label="可发布服务" :value="publishableCount" foot="后端判定可执行" />
      <MetricCard label="已勾选服务" :value="selectedIds.length" foot="实际勾选数量" />
    </div>

    <el-card v-loading="loading" shadow="never">
      <div class="toolbar">
        <div class="toolbar-left wrap">
          <el-radio-group v-model="statusFilter">
            <el-radio-button label="ALL">全部</el-radio-button>
            <el-radio-button label="NEED_UPDATE">只看需更新</el-radio-button>
            <el-radio-button label="MISSING_IN_TARGET">只看服务部署</el-radio-button>
            <el-radio-button label="WORKLOAD_ERROR">只看 workload 异常</el-radio-button>
            <el-radio-button label="NOT_PUBLISHABLE">只看不可发布</el-radio-button>
          </el-radio-group>
          <el-input v-model="keyword" placeholder="搜索服务、tag、namespace" clearable />
        </div>
        <div class="top-actions">
          <el-button @click="selectPublishable">批量选择可执行服务</el-button>
          <el-button @click="goCreateByMode('SERVICE_DEPLOYMENT')">创建部署任务</el-button>
          <el-button type="primary" @click="goCreateByMode('SERVICE_RELEASE')">创建发布单</el-button>
        </div>
      </div>
      <ServiceDiffTable v-model:selected-ids="selectedIds" :items="filteredItems" />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MetricCard from '@/components/MetricCard.vue'
import ServiceDiffTable from '@/components/ServiceDiffTable.vue'
import { getBaselineCompare, getBaselineDetail } from '@/api/baselines'
import { listEnvironments } from '@/api/environments'
import { baselineMockData } from '@/api/mockData/baseline'
import { environmentMockData } from '@/api/mockData/environment'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const data = ref({ ...baselineMockData.diffResult })
const baselineDetail = ref({ ...baselineMockData.baselineDetail })
const environments = ref<typeof environmentMockData.environments>([])
const keyword = ref('')
const statusFilter = ref('ALL')
const selectedIds = ref<string[]>([])
const baselineId = computed(() => String(route.query.baselineId || data.value.sourceBaselineId || 'BL-20260607-0001'))
const targetEnvironmentId = computed(() => String(route.query.targetEnvironmentId || data.value.targetEnvironmentId || ''))
const publishableCount = computed(() => data.value.summary.publishable ?? data.value.items.filter((item) => item.publishable).length)
const baselineSummary = computed(() => `${baselineDetail.value.sourceEnvironmentName} / ${baselineDetail.value.serviceCount} 服务 / ${baselineDetail.value.status}`)
const targetEnvironment = computed(() => environments.value.find((item) => item.id === targetEnvironmentId.value))
const targetSummary = computed(() =>
  targetEnvironment.value
    ? `${targetEnvironment.value.name} / ${targetEnvironment.value.code}`
    : `目标环境 ${targetEnvironmentId.value || data.value.targetEnvironmentId}`,
)

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return data.value.items.filter((item) => {
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

function goCreateByMode(mode: 'SERVICE_RELEASE' | 'SERVICE_DEPLOYMENT') {
  const selectedItems = data.value.items.filter((item) => selectedIds.value.includes(item.serviceId))
  const expectedStatus = mode === 'SERVICE_DEPLOYMENT' ? 'MISSING_IN_TARGET' : 'NEED_UPDATE'
  const targetItems = selectedItems.filter((item) => item.diffStatus === expectedStatus)

  if (targetItems.length === 0) {
    ElMessage.warning(mode === 'SERVICE_DEPLOYMENT' ? '请先选择待部署服务' : '请先选择待发版服务')
    return
  }

  if (targetItems.length !== selectedItems.length) {
    ElMessage.warning(mode === 'SERVICE_DEPLOYMENT'
      ? '部署任务只能包含目标环境缺失的服务，请重新勾选'
      : '发布单只能包含需更新服务，请重新勾选')
    return
  }

  router.push({
    path: '/releases/create',
    query: {
      baselineId: data.value.sourceBaselineId,
      targetEnvironmentId: targetEnvironmentId.value || data.value.targetEnvironmentId,
      mode,
      serviceIds: targetItems.map((item) => item.serviceId).join(','),
    },
  })
}

async function loadEnvironments() {
  try {
    environments.value = await listEnvironments()
  } catch {
    environments.value = [...environmentMockData.environments]
  }
}

async function loadCompare() {
  loading.value = true
  try {
    const [detail, result] = await Promise.all([
      getBaselineDetail(baselineId.value),
      getBaselineCompare(baselineId.value, targetEnvironmentId.value),
    ])
    baselineDetail.value = detail
    data.value = result
  } catch {
    ElMessage.warning('加载差异对比失败，已显示本地示例数据')
    baselineDetail.value = { ...baselineMockData.baselineDetail }
    data.value = { ...baselineMockData.diffResult }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadEnvironments()
  await loadCompare()
})

watch(() => route.fullPath, loadCompare)
</script>
