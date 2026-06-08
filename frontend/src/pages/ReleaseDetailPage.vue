<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>发布详情：{{ release.id }}</h1>
        <p>基线差异发布，{{ release.sourceBaselineId }} 到 {{ release.targetEnvironmentName }}。</p>
      </div>
      <div class="top-actions">
        <el-button disabled>暂停</el-button>
        <el-button type="danger" :loading="retrying" :disabled="!canRetryRelease" @click="handleRetryRelease">失败重试</el-button>
        <el-button :loading="rollingBack" :disabled="!canRollbackRelease" @click="handleRollbackRelease">执行回滚</el-button>
        <el-button type="primary" :disabled="!releaseReport" @click="reportVisible = true">查看发布报告</el-button>
      </div>
    </div>

    <div v-loading="loading" class="metric-grid six">
      <MetricCard label="整体进度" :value="`${displayProgress}%`" :foot="progressFoot" />
      <MetricCard label="当前阶段" :value="currentStepName" :foot="runtimeStatusText" :tone="currentStepTone" />
      <MetricCard label="已完成步骤" :value="`${completedStepCount}/${release.steps.length}`" :foot="completedStepFoot" :tone="completedStepTone" />
      <MetricCard label="失败服务" :value="release.failures.length" :foot="failureFoot" :tone="release.failures.length > 0 ? 'bad' : 'good'" />
      <MetricCard label="执行模式" :value="executionModeLabel" :foot="agentTaskHint" :tone="executionModeTone" />
      <MetricCard label="执行 Agent" :value="agentHealthLabel" :foot="release.agentName" :tone="agentHealthTone" />
    </div>

    <div v-loading="loading" class="two-col">
      <DeployStepPanel title="发布步骤" :status="panelStatus" :steps="displaySteps" :active-step-name="currentStepName" />
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>Agent 执行日志</strong><el-button link type="primary">复制日志</el-button></div></template>
        <LogTerminal :title="`${release.agentName} / ${logTitleId}`" :logs="agentLogs" :badge="agentBadge" />
      </el-card>
    </div>

    <el-card v-loading="loading" shadow="never">
      <template #header><strong>失败定位建议</strong></template>
      <el-table :data="release.failures" class="wide-table">
        <el-table-column prop="serviceName" label="服务" min-width="150" />
        <el-table-column prop="reason" label="失败原因" min-width="160" />
        <el-table-column prop="suggestion" label="建议处理动作" min-width="320" />
        <el-table-column label="操作" fixed="right" width="120">
          <template #default="{ row }"><el-button link type="primary" @click="openFailure(row)">详情</el-button></template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card v-loading="loading" shadow="never">
      <template #header><strong>执行记录</strong></template>
      <el-table :data="actionRecords" class="wide-table">
        <el-table-column prop="occurredAt" label="时间" min-width="180" />
        <el-table-column prop="action" label="动作" min-width="180" />
        <el-table-column prop="operator" label="执行人" min-width="120" />
        <el-table-column label="状态" min-width="120">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="message" label="说明" min-width="320" />
      </el-table>
    </el-card>

    <ServiceFailureDrawer v-model:visible="drawerVisible" :failure="activeFailure" />

    <el-dialog v-model="reportVisible" title="发布报告" width="640px">
      <div v-if="releaseReport" class="report-grid">
        <div class="report-item">
          <span>生成时间</span>
          <strong>{{ releaseReport.generatedAt }}</strong>
        </div>
        <div class="report-item">
          <span>操作人</span>
          <strong>{{ releaseReport.operator }}</strong>
        </div>
        <div class="report-item">
          <span>成功服务</span>
          <strong>{{ releaseReport.successServiceCount }}</strong>
        </div>
        <div class="report-item">
          <span>失败服务</span>
          <strong>{{ releaseReport.failedServiceCount }}</strong>
        </div>
        <div class="report-item">
          <span>人工确认次数</span>
          <strong>{{ releaseReport.manualConfirmCount }}</strong>
        </div>
        <div class="report-item">
          <span>回滚建议</span>
          <StatusTag :status="releaseReport.rollbackRecommended ? 'FAILED' : 'SUCCESS'" />
        </div>
      </div>
      <p v-if="releaseReport" class="report-summary">{{ releaseReport.summary }}</p>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import DeployStepPanel from '@/components/DeployStepPanel.vue'
import LogTerminal from '@/components/LogTerminal.vue'
import MetricCard from '@/components/MetricCard.vue'
import ServiceFailureDrawer from '@/components/ServiceFailureDrawer.vue'
import StatusTag from '@/components/StatusTag.vue'
import { getAgentTaskStatus, type AgentTaskStatus } from '@/api/agentTasks'
import { getReleaseDetail, retryRelease, rollbackRelease } from '@/api/releases'
import { releaseMockData } from '@/api/mockData/release'

type Failure = (typeof releaseMockData.releaseDetail.failures)[number]

const route = useRoute()
const loading = ref(false)
const release = ref({ ...releaseMockData.releaseDetail })
const agentStatus = ref<AgentTaskStatus | null>(null)
const drawerVisible = ref(false)
const activeFailure = ref<Failure | null>(null)
const reportVisible = ref(false)
const retrying = ref(false)
const rollingBack = ref(false)
const releaseId = computed(() => String(route.params.id || release.value.id))
const routeAgentTaskId = computed(() => (route.query.agentTaskId ? String(route.query.agentTaskId) : ''))
const agentTaskId = computed(() => release.value.agentTaskId || routeAgentTaskId.value)
const logTitleId = computed(() => agentTaskId.value || release.value.id)
const runtimeState = computed(() => agentStatus.value?.status?.status ?? '')
const runtimeStep = computed(() => agentStatus.value?.status?.step?.trim() ?? '')
const statusToneMap: Record<string, 'good' | 'warn' | 'bad'> = {
  SUCCESS: 'good',
  RUNNING: 'warn',
  PENDING: 'warn',
  PENDING_CONFIRM: 'warn',
  WAITING_CONFIRM: 'warn',
  FAILED: 'bad',
  PARTIAL_FAILED: 'bad',
  CANCELLED: 'bad',
}
const runtimeProgressMap: Record<string, number> = {
  PENDING: 10,
  PENDING_CONFIRM: 20,
  RUNNING: 72,
  WAITING_CONFIRM: 85,
  SUCCESS: 100,
  FAILED: 100,
  PARTIAL_FAILED: 100,
  CANCELLED: 100,
}
const runtimeLabelMap: Record<string, string> = {
  PENDING: '待执行',
  PENDING_CONFIRM: '待确认',
  RUNNING: '执行中',
  WAITING_CONFIRM: '等待确认',
  SUCCESS: '执行成功',
  FAILED: '执行失败',
  PARTIAL_FAILED: '部分失败',
  CANCELLED: '已取消',
}
const agentLogs = computed(() => {
  const logs = agentStatus.value?.logs ?? []
  return logs.length > 0 ? logs : release.value.logs
})
const agentBadge = computed(() => agentStatus.value?.status?.status ?? (agentStatus.value?.enabled === false ? 'disabled' : 'live'))
const displayProgress = computed(() => runtimeProgressMap[runtimeState.value] ?? release.value.progress)
const currentStepName = computed(() => {
  if (runtimeStep.value) return runtimeStep.value
  return (
    release.value.steps.find((item) => ['RUNNING', 'WAITING_CONFIRM', 'FAILED', 'PARTIAL_FAILED', 'PENDING'].includes(item.status))?.name ||
    release.value.steps.at(-1)?.name ||
    '等待调度'
  )
})
const progressFoot = computed(() => runtimeStep.value ? `正在执行 ${runtimeStep.value}` : '按详情快照展示')
const runtimeStatusText = computed(() => {
  if (agentStatus.value?.enabled === false) return '轮询已关闭，显示静态详情'
  return runtimeLabelMap[runtimeState.value] ?? release.value.status
})
const currentStepTone = computed(() => statusToneMap[runtimeState.value] ?? 'warn')
const displaySteps = computed(() =>
  release.value.steps.map((step, index) => {
    const runtimeMatched = runtimeStep.value && step.name === runtimeStep.value
    if (runtimeMatched) {
      return {
        ...step,
        status: runtimeState.value || step.status,
        message: step.message || '来自 Agent 的实时步骤状态',
      }
    }
    if (!runtimeStep.value) return step
    const runtimeIndex = release.value.steps.findIndex((item) => item.name === runtimeStep.value)
    if (runtimeIndex === -1) return step
    if (index < runtimeIndex && !['FAILED', 'PARTIAL_FAILED', 'WAITING_CONFIRM'].includes(step.status)) {
      return {
        ...step,
        status: 'SUCCESS',
      }
    }
    if (index > runtimeIndex && step.status === 'RUNNING') {
      return {
        ...step,
        status: 'PENDING',
      }
    }
    return step
  }),
)
const completedStepCount = computed(() => displaySteps.value.filter((item) => item.status === 'SUCCESS').length)
const actionRecords = computed(() => release.value.actionRecords ?? [])
const releaseReport = computed(() => release.value.report ?? null)
const completedStepFoot = computed(() => {
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return '当前卡在人工确认'
  if (runtimeStep.value) return `当前执行：${runtimeStep.value}`
  return '按详情快照展示'
})
const completedStepTone = computed(() => (completedStepCount.value === release.value.steps.length ? 'good' : 'warn'))
const failureFoot = computed(() => {
  if (runtimeState.value === 'FAILED' || runtimeState.value === 'PARTIAL_FAILED') return '存在失败步骤，优先处理失败服务'
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return '等待人工确认后继续执行'
  return release.value.failures.length > 0 ? '可重试并查看失败建议' : '当前无失败服务'
})
const executionModeLabel = computed(() => {
  const type = agentStatus.value?.status?.type
  if (type === 'release') return '实时 Agent'
  return agentStatus.value?.enabled === false ? '静态快照' : agentTaskId.value ? 'Agent 编排' : '静态回放'
})
const executionModeTone = computed(() => (agentStatus.value?.enabled === false ? 'warn' : agentTaskId.value ? 'good' : 'warn'))
const agentTaskHint = computed(() => agentTaskId.value || agentStatus.value?.message || '未关联实时任务')
const agentHealthLabel = computed(() => {
  if (agentStatus.value?.enabled === false) return '离线回放'
  if (runtimeState.value === 'FAILED' || runtimeState.value === 'CANCELLED') return '需处理'
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return '待确认'
  return agentTaskId.value ? '在线' : '未知'
})
const agentHealthTone = computed(() => {
  if (agentStatus.value?.enabled === false) return 'warn'
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return 'warn'
  return runtimeState.value === 'FAILED' || runtimeState.value === 'CANCELLED' ? 'bad' : 'good'
})
const panelStatus = computed(() => runtimeState.value || release.value.status)
const effectiveReleaseStatus = computed(() => runtimeState.value || release.value.status)
const canRetryRelease = computed(() => ['FAILED', 'PARTIAL_FAILED', 'CANCELLED'].includes(effectiveReleaseStatus.value) && !retrying.value && !rollingBack.value)
const canRollbackRelease = computed(
  () => ['RUNNING', 'WAITING_CONFIRM', 'PENDING_CONFIRM', 'FAILED', 'PARTIAL_FAILED'].includes(effectiveReleaseStatus.value) && !retrying.value && !rollingBack.value,
)
let pollingTimer: number | undefined

function openFailure(row: Failure) {
  activeFailure.value = row
  drawerVisible.value = true
}

async function loadRelease() {
  loading.value = true
  try {
    release.value = await getReleaseDetail(releaseId.value)
  } catch {
    ElMessage.warning('加载发布详情失败，已显示本地示例数据')
    release.value = { ...releaseMockData.releaseDetail }
  } finally {
    loading.value = false
  }
}

async function reloadDetailState() {
  await loadRelease()
  await pollAgentStatus()
}

async function handleRetryRelease() {
  if (!canRetryRelease.value) return
  retrying.value = true
  try {
    const result = await retryRelease(releaseId.value)
    ElMessage.success(result.message || '已提交失败重试')
    await reloadDetailState()
  } catch {
    ElMessage.warning('提交失败重试失败，请稍后重试')
  } finally {
    retrying.value = false
  }
}

async function handleRollbackRelease() {
  if (!canRollbackRelease.value) return
  rollingBack.value = true
  try {
    const result = await rollbackRelease(releaseId.value)
    ElMessage.success(result.message || '已提交回滚任务')
    await reloadDetailState()
  } catch {
    ElMessage.warning('提交回滚失败，请稍后重试')
  } finally {
    rollingBack.value = false
  }
}

async function pollAgentStatus() {
  if (!agentTaskId.value) {
    agentStatus.value = {
      enabled: false,
      message: '未关联实时 Agent 任务，展示详情快照',
      logs: [],
    }
    return
  }
  try {
    agentStatus.value = await getAgentTaskStatus(agentTaskId.value)
  } catch {
    agentStatus.value = null
  }
}

onMounted(async () => {
  await loadRelease()
  await pollAgentStatus()
  pollingTimer = window.setInterval(pollAgentStatus, 2000)
})

watch(
  () => route.fullPath,
  async () => {
    await loadRelease()
    await pollAgentStatus()
  },
)

onUnmounted(() => {
  if (pollingTimer) {
    window.clearInterval(pollingTimer)
  }
})
</script>

<style scoped>
.report-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.report-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.report-item span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.report-summary {
  margin: 16px 0 0;
  color: var(--el-text-color-regular);
  line-height: 1.6;
}
</style>
