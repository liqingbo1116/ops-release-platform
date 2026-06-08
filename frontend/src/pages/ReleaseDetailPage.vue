<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>发布详情：{{ release.id }}</h1>
        <p>基线差异发布，{{ release.sourceBaselineId }} 到 {{ release.targetEnvironmentName }}。</p>
      </div>
      <div class="top-actions">
        <el-button disabled>暂停</el-button>
        <el-button type="danger" disabled>失败重试</el-button>
        <el-button type="primary" disabled>生成发布报告</el-button>
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

    <ServiceFailureDrawer v-model:visible="drawerVisible" :failure="activeFailure" />
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
import { getAgentTaskStatus, type AgentTaskStatus } from '@/api/agentTasks'
import { getReleaseDetail } from '@/api/releases'
import { releaseMockData } from '@/api/mockData/release'

type Failure = (typeof releaseMockData.releaseDetail.failures)[number]

const route = useRoute()
const loading = ref(false)
const release = ref({ ...releaseMockData.releaseDetail })
const agentStatus = ref<AgentTaskStatus | null>(null)
const drawerVisible = ref(false)
const activeFailure = ref<Failure | null>(null)
const releaseId = computed(() => String(route.params.id || release.value.id))
const agentTaskId = computed(() => (route.query.agentTaskId ? String(route.query.agentTaskId) : ''))
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
const completedStepCount = computed(() => release.value.steps.filter((item) => item.status === 'SUCCESS').length)
const completedStepFoot = computed(() => runtimeStep.value ? `当前执行：${runtimeStep.value}` : '按详情快照展示')
const completedStepTone = computed(() => (completedStepCount.value === release.value.steps.length ? 'good' : 'warn'))
const failureFoot = computed(() => (release.value.failures.length > 0 ? '可重试并查看失败建议' : '当前无失败服务'))
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
  return agentTaskId.value ? '在线' : '未知'
})
const agentHealthTone = computed(() => {
  if (agentStatus.value?.enabled === false) return 'warn'
  return runtimeState.value === 'FAILED' || runtimeState.value === 'CANCELLED' ? 'bad' : 'good'
})
const panelStatus = computed(() => runtimeState.value || release.value.status)
const displaySteps = computed(() =>
  release.value.steps.map((step) => {
    if (runtimeStep.value && step.name === runtimeStep.value) {
      return {
        ...step,
        status: runtimeState.value || step.status,
        message: step.message || '来自 Agent 的实时步骤状态',
      }
    }
    return step
  }),
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
