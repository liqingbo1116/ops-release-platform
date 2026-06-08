<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>部署任务详情：{{ deploy.id }}</h1>
        <p>{{ deploy.targetEnvironmentName }} 初始化部署，来源 {{ deploy.source }}，当前进度 {{ displayProgress }}%。</p>
      </div>
      <div class="top-actions">
        <el-button disabled>暂停</el-button>
        <el-button disabled>跳过当前步骤</el-button>
        <el-button type="danger" disabled>重试失败步骤</el-button>
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
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import DeployStepPanel from '@/components/DeployStepPanel.vue'
import LogTerminal from '@/components/LogTerminal.vue'
import MetricCard from '@/components/MetricCard.vue'
import { getAgentTaskStatus, type AgentTaskStatus } from '@/api/agentTasks'
import { getDeployTaskDetail } from '@/api/deployTasks'
import { deployMockData } from '@/api/mockData/deploy'

const route = useRoute()
const loading = ref(false)
const deploy = ref({ ...deployMockData.deployDetail })
const agentStatus = ref<AgentTaskStatus | null>(null)
const deployTaskId = computed(() => String(route.params.id || deploy.value.id))
const agentTaskId = computed(() => (route.query.agentTaskId ? String(route.query.agentTaskId) : ''))
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
  return logs.length > 0 ? logs : deploy.value.logs
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
const completedStepCount = computed(() => deploy.value.steps.filter((item) => item.status === 'SUCCESS').length)
const progressFoot = computed(() => `${completedStepCount.value}/${deploy.value.steps.length} 步完成`)
const stepSummaryFoot = computed(() => {
  const manualStepCount = deploy.value.steps.filter((item) => item.type === 'MANUAL_CONFIRM').length
  return `含 ${manualStepCount} 个人工确认`
})
const failureCount = computed(() => agentLogs.value.filter((item) => item.includes('ERROR') || item.includes('WARN')).length)
const failureFoot = computed(() => {
  if (runtimeState.value === 'FAILED') return '当前任务失败，需处理后重试'
  if (failureCount.value > 0) return '日志存在告警，建议关注当前步骤'
  return '当前无失败记录'
})
const agentHealthLabel = computed(() => {
  if (agentStatus.value?.enabled === false) return '离线回放'
  if (runtimeState.value === 'FAILED' || runtimeState.value === 'CANCELLED') return '需处理'
  return agentTaskId.value ? '在线' : '未知'
})
const agentTaskHint = computed(() => agentTaskId.value || agentStatus.value?.message || '未关联任务 ID')
const agentHealthTone = computed(() => {
  if (agentStatus.value?.enabled === false) return 'warn'
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
const displaySteps = computed(() =>
  deploy.value.steps.map((step) => {
    if (runtimeStep.value && step.name === runtimeStep.value) {
      return {
        ...step,
        status: runtimeState.value || step.status,
      }
    }
    return step
  }),
)
let pollingTimer: number | undefined

async function loadDeployTask() {
  loading.value = true
  try {
    deploy.value = await getDeployTaskDetail(deployTaskId.value)
  } catch {
    ElMessage.warning('加载部署任务详情失败，已显示本地示例数据')
    deploy.value = { ...deployMockData.deployDetail }
  } finally {
    loading.value = false
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
