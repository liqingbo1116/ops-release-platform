<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>部署任务详情：{{ deploy.id }}</h1>
        <p>{{ deploy.targetEnvironmentName }} 初始化部署，来源 {{ deploy.source }}，当前进度 {{ displayProgress }}%。</p>
      </div>
      <div class="top-actions">
        <el-button disabled>暂停</el-button>
        <el-button :loading="skippingStep" :disabled="!canSkipCurrentStep" @click="handleSkipCurrentStep">跳过当前步骤</el-button>
        <el-button type="warning" :loading="confirmingStep" :disabled="!canConfirmCurrentStep" @click="handleConfirmCurrentStep">人工确认继续</el-button>
        <el-button type="danger" :loading="retryingStep" :disabled="!canRetryCurrentStep" @click="handleRetryCurrentStep">重试失败步骤</el-button>
      </div>
    </div>

    <div v-loading="loading" class="metric-grid six">
      <MetricCard label="整体进度" :value="`${displayProgress}%`" :foot="progressFoot" />
      <MetricCard label="当前步骤" :value="currentStepName" :foot="runtimeStatusText" :tone="currentStepTone" />
      <MetricCard label="执行步骤" :value="deploy.steps.length" :foot="stepSummaryFoot" />
      <MetricCard label="脚本步骤" :value="scriptStepCount" />
      <MetricCard label="失败次数" :value="failureCount" :foot="failureFoot" :tone="failureCount > 0 ? 'bad' : 'good'" />
      <MetricCard label="当前 Agent" :value="agentHealthLabel" :foot="agentTaskHint" :tone="agentHealthTone" />
      <MetricCard label="执行模式" :value="executionModeLabel" :foot="durationFoot" :tone="executionModeTone" />
    </div>

    <div v-loading="loading" class="two-col">
      <DeployStepPanel title="部署步骤编排" :status="panelStatus" :steps="displaySteps" :active-step-name="currentStepName" />
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>当前步骤日志</strong><el-button type="danger" link disabled>重试当前步骤</el-button></div></template>
        <LogTerminal :title="`${logTitleId} / ${agentStep}`" :logs="agentLogs" :badge="agentBadge" />
      </el-card>
    </div>

    <el-card v-loading="loading" shadow="never">
      <template #header><strong>审计与影响范围</strong></template>
      <div class="audit-grid">
        <div class="audit-item">
          <span>操作人</span>
          <strong>{{ auditSummary.operator || '-' }}</strong>
        </div>
        <div class="audit-item">
          <span>目标环境</span>
          <strong>{{ auditSummary.targetEnvironmentName || deploy.targetEnvironmentName }}</strong>
        </div>
        <div class="audit-item">
          <span>执行结果</span>
          <StatusTag :status="auditSummary.result || effectiveDeployStatus" />
        </div>
        <div class="audit-item">
          <span>失败步骤</span>
          <strong>{{ auditSummary.failedStep || '无' }}</strong>
        </div>
        <div class="audit-item wide">
          <span>影响服务</span>
          <strong>{{ affectedServiceLabel }}</strong>
        </div>
        <div class="audit-item wide">
          <span>最后动作</span>
          <strong>{{ auditSummary.lastAction || '-' }} / {{ auditSummary.lastActionAt || '-' }}</strong>
        </div>
      </div>
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
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import DeployStepPanel from '@/components/DeployStepPanel.vue'
import LogTerminal from '@/components/LogTerminal.vue'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { getAgentTaskStatus, type AgentTaskStatus } from '@/api/agentTasks'
import { confirmDeployStep, getDeployTaskDetail, retryDeployStep, skipDeployStep, type DeployDetail } from '@/api/deployTasks'

const route = useRoute()
const loading = ref(false)
const emptyDeployDetail = (id: string): DeployDetail => ({
  id,
  productName: '',
  targetEnvironmentName: '',
  source: '',
  status: 'UNKNOWN',
  progress: 0,
  steps: [],
  logs: [],
  actionRecords: [],
})
const deploy = ref<DeployDetail>(emptyDeployDetail(String(route.params.id || '')))
const agentStatus = ref<AgentTaskStatus | null>(null)
const retryingStep = ref(false)
const skippingStep = ref(false)
const confirmingStep = ref(false)
const deployTaskId = computed(() => String(route.params.id || deploy.value.id))
const routeAgentTaskId = computed(() => (route.query.agentTaskId ? String(route.query.agentTaskId) : ''))
const agentTaskId = computed(() => deploy.value.agentTaskId || routeAgentTaskId.value)
const logTitleId = computed(() => agentTaskId.value || deploy.value.id)
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
  PENDING: 12,
  PENDING_CONFIRM: 24,
  RUNNING: 58,
  WAITING_CONFIRM: 72,
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
const agentStep = computed(() => {
  if (runtimeStep.value) return runtimeStep.value
  return (
    deploy.value.steps.find((item) => ['RUNNING', 'WAITING_CONFIRM', 'FAILED', 'PARTIAL_FAILED', 'PENDING'].includes(item.status))?.name ||
    deploy.value.steps.at(-1)?.name ||
    '等待调度'
  )
})
const agentLogs = computed(() => {
  const logs = agentStatus.value?.logs ?? []
  return logs.length > 0 ? logs : (deploy.value.logs ?? [])
})
const agentBadge = computed(() => agentStatus.value?.status?.status ?? (agentStatus.value?.enabled === false ? 'disabled' : 'retry #1'))
const scriptStepCount = computed(() => deploy.value.steps.filter((item) => ['SHELL', 'SQL'].includes(item.type)).length)
const displayProgress = computed(() => runtimeProgressMap[runtimeState.value] ?? deploy.value.progress)
const currentStepName = computed(() => agentStep.value)
const runtimeStatusText = computed(() => {
  if (agentStatus.value?.enabled === false) return '轮询已关闭，显示静态详情'
  return runtimeLabelMap[runtimeState.value] ?? deploy.value.status
})
const currentStepTone = computed(() => statusToneMap[runtimeState.value] ?? 'warn')
const displaySteps = computed(() => {
  const runtimeIndex = runtimeStep.value ? deploy.value.steps.findIndex((item) => item.name === runtimeStep.value) : -1
  return deploy.value.steps.map((step, index) => {
    if (runtimeStep.value && step.name === runtimeStep.value) {
      return {
        ...step,
        status: runtimeState.value || step.status,
      }
    }
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
  })
})
const completedStepCount = computed(() => displaySteps.value.filter((item) => item.status === 'SUCCESS').length)
const actionRecords = computed(() => deploy.value.actionRecords ?? [])
const auditSummary = computed(() => deploy.value.auditSummary ?? {
  operator: actionRecords.value[0]?.operator || '',
  targetEnvironmentName: deploy.value.targetEnvironmentName,
  affectedServices: [],
  result: effectiveDeployStatus.value,
  failedStep: displaySteps.value.find((item) => ['FAILED', 'PARTIAL_FAILED'].includes(item.status))?.name || '',
  lastAction: actionRecords.value.at(-1)?.action || '',
  lastActionAt: actionRecords.value.at(-1)?.occurredAt || '',
})
const affectedServiceLabel = computed(() => {
  const services = auditSummary.value.affectedServices ?? []
  return services.length > 0 ? services.join('、') : '未记录'
})
const progressFoot = computed(() => `${completedStepCount.value}/${deploy.value.steps.length} 步完成`)
const stepSummaryFoot = computed(() => {
  const manualStepCount = deploy.value.steps.filter((item) => item.type === 'MANUAL_CONFIRM').length
  return `含 ${manualStepCount} 个人工确认`
})
const failureCount = computed(() => agentLogs.value.filter((item) => item.includes('ERROR') || item.includes('WARN')).length)
const failureFoot = computed(() => {
  if (runtimeState.value === 'FAILED') return '当前任务失败，需处理后重试'
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return '等待人工确认后继续执行'
  if (failureCount.value > 0) return '日志存在告警，建议关注当前步骤'
  return '当前无失败记录'
})
const agentHealthLabel = computed(() => {
  if (agentStatus.value?.enabled === false) return '离线回放'
  if (runtimeState.value === 'FAILED' || runtimeState.value === 'CANCELLED') return '需处理'
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return '待确认'
  return agentTaskId.value ? '在线' : '未知'
})
const agentTaskHint = computed(() => agentTaskId.value || agentStatus.value?.message || '未关联任务 ID')
const agentHealthTone = computed(() => {
  if (agentStatus.value?.enabled === false) return 'warn'
  if (runtimeState.value === 'WAITING_CONFIRM' || runtimeState.value === 'PENDING_CONFIRM') return 'warn'
  return runtimeState.value === 'FAILED' || runtimeState.value === 'CANCELLED' ? 'bad' : 'good'
})
const executionModeLabel = computed(() => {
  const type = agentStatus.value?.status?.type
  if (type === 'deploy') return '实时 Agent'
  return agentStatus.value?.enabled === false ? '静态快照' : agentTaskId.value ? 'Agent 编排' : '静态回放'
})
const executionModeTone = computed(() => (agentStatus.value?.enabled === false ? 'warn' : agentTaskId.value ? 'good' : 'warn'))
const durationFoot = computed(() => agentStatus.value?.status?.updatedAt ? `最近更新 ${agentStatus.value.status?.updatedAt}` : '等待下一次轮询')
const panelStatus = computed(() => runtimeState.value || deploy.value.status)
const effectiveDeployStatus = computed(() => runtimeState.value || deploy.value.status)
const currentStep = computed(() => displaySteps.value.find((item) => item.name === currentStepName.value) || displaySteps.value.find((item) => item.status !== 'SUCCESS'))
const currentStepId = computed(() => {
  const step = currentStep.value as ({ id?: string | number; order?: string | number } & Record<string, unknown>) | undefined
  return String(step?.id ?? step?.order ?? '')
})
const canRetryCurrentStep = computed(() => ['FAILED', 'PARTIAL_FAILED'].includes(effectiveDeployStatus.value) && !!currentStepId.value && !retryingStep.value && !skippingStep.value && !confirmingStep.value)
const canSkipCurrentStep = computed(() => ['RUNNING', 'FAILED', 'WAITING_CONFIRM'].includes(effectiveDeployStatus.value) && !!currentStepId.value && !retryingStep.value && !skippingStep.value && !confirmingStep.value)
const canConfirmCurrentStep = computed(
  () => ['WAITING_CONFIRM', 'PENDING_CONFIRM'].includes(effectiveDeployStatus.value) && !!currentStepId.value && !retryingStep.value && !skippingStep.value && !confirmingStep.value,
)
let pollingTimer: number | undefined

async function loadDeployTask() {
  loading.value = true
  try {
    deploy.value = await getDeployTaskDetail(deployTaskId.value)
  } catch {
    ElMessage.error('加载部署任务详情失败，请检查任务是否存在或后端接口是否正常')
    deploy.value = emptyDeployDetail(deployTaskId.value)
  } finally {
    loading.value = false
  }
}

async function reloadDetailState() {
  await loadDeployTask()
  await pollAgentStatus()
}

async function handleRetryCurrentStep() {
  if (!canRetryCurrentStep.value || !currentStepId.value) return
  retryingStep.value = true
  try {
    const result = await retryDeployStep(deployTaskId.value, currentStepId.value)
    ElMessage.success(result.message || '已提交步骤重试')
    await reloadDetailState()
  } catch {
    ElMessage.warning('提交步骤重试失败，请稍后重试')
  } finally {
    retryingStep.value = false
  }
}

async function handleSkipCurrentStep() {
  if (!canSkipCurrentStep.value || !currentStepId.value) return
  skippingStep.value = true
  try {
    const result = await skipDeployStep(deployTaskId.value, currentStepId.value)
    ElMessage.success(result.message || '已提交步骤跳过')
    await reloadDetailState()
  } catch {
    ElMessage.warning('提交步骤跳过失败，请稍后重试')
  } finally {
    skippingStep.value = false
  }
}

async function handleConfirmCurrentStep() {
  if (!canConfirmCurrentStep.value || !currentStepId.value) return
  confirmingStep.value = true
  try {
    const result = await confirmDeployStep(deployTaskId.value, currentStepId.value)
    ElMessage.success(result.message || '已提交人工确认')
    await reloadDetailState()
  } catch {
    ElMessage.warning('提交人工确认失败，请稍后重试')
  } finally {
    confirmingStep.value = false
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
  await loadDeployTask()
  await pollAgentStatus()
  pollingTimer = window.setInterval(pollAgentStatus, 2000)
})

watch(
  () => route.fullPath,
  async () => {
    await loadDeployTask()
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
.audit-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.audit-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.audit-item.wide {
  grid-column: span 2;
}

.audit-item span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.audit-item strong {
  overflow-wrap: anywhere;
}

@media (max-width: 900px) {
  .audit-grid {
    grid-template-columns: 1fr;
  }

  .audit-item.wide {
    grid-column: span 1;
  }
}
</style>
