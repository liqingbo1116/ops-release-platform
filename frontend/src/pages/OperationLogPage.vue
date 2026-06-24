<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>审计日志</h1>
        <p>记录用户登录、服务纳管、发版部署等关键操作，实际资源状态仍以产品环境为准。</p>
      </div>
    </div>

    <el-card shadow="never" class="log-card">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="filters.keyword" placeholder="搜索动作、对象、说明" clearable class="keyword-input" />
          <el-select v-model="filters.environmentId" placeholder="全部产品" clearable filterable class="filter-select">
            <el-option v-for="item in products" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
          <el-select v-model="filters.resourceType" placeholder="全部对象" clearable class="filter-select">
            <el-option label="Agent" value="AGENT" />
            <el-option label="服务" value="SERVICE" />
            <el-option label="工作负载" value="WORKLOAD" />
            <el-option label="用户" value="USER" />
            <el-option label="发布单" value="RELEASE" />
            <el-option label="部署任务" value="DEPLOY" />
          </el-select>
        </div>
        <el-button :loading="loading" @click="loadData">刷新</el-button>
      </div>

      <el-table v-loading="loading" :data="logs" stripe>
        <el-table-column label="时间" prop="createdAt" width="150">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="事件" prop="action" width="150">
          <template #default="{ row }">
            <el-tag effect="plain" :type="actionTagType(row.action)">{{ actionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="对象" min-width="220">
          <template #default="{ row }">
            <div class="object-cell">
              <span>{{ resourceTypeLabel(row.resourceType) }}</span>
              <strong>{{ resourceName(row) }}</strong>
              <code>{{ resourceMeta(row) }}</code>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="项目 / 产品" min-width="190">
          <template #default="{ row }">
            <div class="product-cell">
              <span>{{ projectName(row) }}</span>
              <code>{{ productName(row) }}</code>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="结果" prop="result" width="100">
          <template #default="{ row }">
            <el-tag :type="resultTagType(row.result)" effect="plain">{{ resultLabel(row.result) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="说明" prop="detail" min-width="320" show-overflow-tooltip />
        <el-table-column label="操作人" prop="operatorName" width="120" />
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { listEnvironments, type EnvironmentInfo } from '@/api/environments'
import { listOperationLogs, type OperationLog } from '@/api/operationLogs'
import { formatDateTime } from '@/utils/format'

const loading = ref(false)
const logs = ref<OperationLog[]>([])
const products = ref<EnvironmentInfo[]>([])
const filters = reactive({
  keyword: '',
  environmentId: '',
  resourceType: '',
})

let timer: number | undefined

watch(
  filters,
  () => {
    window.clearTimeout(timer)
    timer = window.setTimeout(loadData, 250)
  },
  { deep: true },
)

function projectName(row: OperationLog) {
  if (row.projectName) return row.projectName
  if (!row.environmentId) return '未绑定项目'
  return products.value.find((item) => item.id === row.environmentId)?.projectName || '未绑定项目'
}

function productName(row: OperationLog) {
  if (row.productName) return row.productName
  if (!row.environmentId) return '-'
  return products.value.find((item) => item.id === row.environmentId)?.name ?? row.environmentId
}

function resourceName(row: OperationLog) {
  return row.workloadName || row.resourceName || row.resourceId || '-'
}

function resourceMeta(row: OperationLog) {
  const parts = []
  if (row.namespace) parts.push(row.namespace)
  if (row.workloadType) parts.push(row.workloadType)
  if (row.containerName) {
    parts.push(`${row.containerType === 'INIT' ? '初始化容器' : '容器'} ${row.containerName}`)
  }
  return parts.length > 0 ? parts.join(' / ') : row.resourceId
}

function actionLabel(value: string) {
  const labels: Record<string, string> = {
    AGENT_CLAIM: 'Agent 绑定产品',
    SERVICE_ADOPT: '服务纳管',
    SERVICE_UNMANAGE_MANUAL: '手动解除纳管',
    SERVICE_AUTO_UNMANAGE: '自动解除纳管',
    USER_LOGIN: '用户登录',
    RELEASE_CREATE: '创建发布单',
    DEPLOY_CREATE: '创建部署任务',
  }
  return labels[value] ?? value
}

function actionTagType(value: string) {
  if (value.includes('AUTO')) return 'warning'
  if (value.includes('UNMANAGE')) return 'info'
  return 'primary'
}

function resourceTypeLabel(value: string) {
  const labels: Record<string, string> = {
    AGENT: 'Agent',
    SERVICE: '服务',
    WORKLOAD: '工作负载',
    USER: '用户',
    RELEASE: '发布单',
    DEPLOY: '部署任务',
  }
  return labels[value] ?? value
}

function resultLabel(value: string) {
  const labels: Record<string, string> = {
    SUCCESS: '成功',
    FAILED: '失败',
  }
  return labels[value] ?? value
}

function resultTagType(value: string) {
  return value === 'FAILED' ? 'danger' : 'success'
}

async function loadData() {
  loading.value = true
  try {
    logs.value = await listOperationLogs({
      keyword: filters.keyword.trim() || undefined,
      environmentId: filters.environmentId || undefined,
      resourceType: filters.resourceType || undefined,
    })
  } finally {
    loading.value = false
  }
}

async function bootstrap() {
  loading.value = true
  try {
    const [productItems, logItems] = await Promise.all([listEnvironments(), listOperationLogs()])
    products.value = productItems
    logs.value = logItems
  } finally {
    loading.value = false
  }
}

onMounted(bootstrap)
</script>

<style scoped>
.log-card {
  overflow: hidden;
}

.keyword-input {
  width: 260px;
}

.filter-select {
  width: 180px;
}

.object-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.object-cell strong {
  overflow: hidden;
  color: #111827;
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.object-cell code {
  overflow: hidden;
  color: #64748b;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.product-cell span,
.product-cell code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-cell code {
  color: #64748b;
  font-size: 12px;
}

@media (max-width: 960px) {
  .toolbar {
    align-items: stretch;
  }

  .toolbar-left,
  .keyword-input,
  .filter-select {
    width: 100%;
  }
}
</style>
