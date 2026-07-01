<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>服务部署任务</h1>
        <p>面向目标环境缺失服务的首次部署，按来源基线、缺失服务、Agent 执行状态和下一步动作跟踪。</p>
      </div>
      <el-button type="primary" :loading="loading" @click="loadRows">刷新</el-button>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索部署任务、来源基线、缺失服务、环境、Agent" clearable />
        <el-button>批量重试失败步骤</el-button>
      </div>
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="id" label="任务" min-width="160" />
        <el-table-column prop="productName" label="项目" min-width="110" />
        <el-table-column prop="targetEnvironmentName" label="目标环境" min-width="160" />
        <el-table-column label="来源基线" min-width="180">
          <template #default="{ row }">
            <div class="deploy-source">
              <strong>{{ row.sourceBaselineId || row.source }}</strong>
              <span>缺失服务首次部署</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="缺失服务" min-width="220">
          <template #default="{ row }">
            <div class="deploy-source">
              <strong>{{ missingServiceText(row) }}</strong>
              <span>{{ serviceNamesText(row) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="currentStep" label="当前步骤" min-width="150" />
        <el-table-column prop="agentName" label="执行 Agent" min-width="160" />
        <el-table-column label="进度" min-width="160">
          <template #default="{ row }"><el-progress :percentage="row.progress" :status="row.status === 'FAILED' ? 'exception' : undefined" /></template>
        </el-table-column>
        <el-table-column label="状态" min-width="110">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="下一步" min-width="220">
          <template #default="{ row }">{{ row.nextAction || nextActionText(row.status) }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="100">
          <template #default="{ row }"><el-button link type="primary" @click="$router.push(`/deploy-tasks/${row.id}`)">查看</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import StatusTag from '@/components/StatusTag.vue'
import { listDeployTasks, type DeployTask } from '@/api/deployTasks'

const keyword = ref('')
const loading = ref(false)
const rows = ref<DeployTask[]>([])
const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((item) =>
    `${item.id} ${item.productName} ${item.targetEnvironmentName} ${item.sourceBaselineId || ''} ${item.source} ${(item.serviceNames || []).join(' ')} ${item.currentStep} ${item.agentName || ''} ${item.agentTaskId || ''} ${item.nextAction || ''}`
      .toLowerCase()
      .includes(q),
  )
})

function missingServiceText(item: DeployTask) {
  const count = item.missingServiceCount ?? item.serviceNames?.length ?? 0
  return count > 0 ? `${count} 个目标缺失服务` : '待确认缺失服务'
}

function serviceNamesText(item: DeployTask) {
  return item.serviceNames?.length ? item.serviceNames.join('、') : '等待差异结果'
}

function nextActionText(status: string) {
  if (status === 'FAILED' || status === 'PARTIAL_FAILED') return '处理失败后重试当前步骤'
  if (status === 'WAITING_CONFIRM' || status === 'PENDING_CONFIRM') return '人工确认后继续'
  if (status === 'SUCCESS') return '首次部署完成'
  return '等待 Agent 执行'
}

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listDeployTasks()
  } catch {
    ElMessage.error('加载部署任务失败，请检查后端接口或数据库')
    rows.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRows)
</script>

<style scoped>
.deploy-source {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.deploy-source span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
