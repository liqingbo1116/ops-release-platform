<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>发版日志</h1>
        <p>{{ release.targetEnvironmentName || '-' }} / {{ affectedServiceLabel }}</p>
      </div>
    </div>

    <div v-loading="loading" class="metric-grid">
      <MetricCard label="状态" :value="releaseStatusLabel" :foot="statusFoot" :tone="releaseStatusTone" />
      <MetricCard label="服务" :value="affectedServiceLabel" :foot="release.targetEnvironmentName || '-'" />
      <MetricCard label="执行" :value="executorLabel" :foot="executorFoot" :tone="executorTone" />
      <MetricCard label="目标镜像" :value="releaseImageLabel" :foot="imageFoot" />
    </div>

    <div v-loading="loading" class="release-brief">
      <span>{{ releaseSourceLabel }}</span>
      <span v-if="release.buildStatus">{{ release.buildStatus }}</span>
      <a v-if="release.buildUrl" :href="release.buildUrl" target="_blank" rel="noreferrer">打开 Jenkins</a>
      <span v-if="release.buildId || agentTaskId">任务 {{ release.buildId || agentTaskId }}</span>
    </div>

    <el-card v-loading="loading" shadow="never">
      <template #header><strong>{{ logPanelTitle }}</strong></template>
      <div class="release-log-scroll">
        <LogTerminal :title="logTerminalTitle" :logs="displayLogs" :badge="logBadge" />
      </div>
    </el-card>

    <el-card v-if="actionRecords.length > 0" v-loading="loading" shadow="never">
      <template #header><strong>执行记录</strong></template>
      <el-table :data="actionRecords" class="wide-table" max-height="220">
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
import LogTerminal from '@/components/LogTerminal.vue'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { getAgentTaskStatus, type AgentTaskStatus } from '@/api/agentTasks'
import { getReleaseDetail, type ReleaseDetail } from '@/api/releases'

const route = useRoute()
const loading = ref(false)
const agentStatus = ref<AgentTaskStatus | null>(null)
const emptyReleaseDetail = (): ReleaseDetail => ({
  id: '',
  type: '',
  sourceBaselineId: '',
  releaseSource: '',
  executionMode: '',
  buildId: '',
  buildStatus: '',
  buildUrl: '',
  jenkinsId: '',
  jenkinsJobName: '',
  jenkinsJobUrl: '',
  imageRepository: '',
  imageTag: '',
  imageDigest: '',
  targetEnvironmentName: '',
  status: 'PENDING',
  progress: 0,
  agentName: '',
  agentTaskId: '',
  serviceIds: [],
  serviceNames: [],
  steps: [],
  failures: [],
  actionRecords: [],
  logs: [],
})
const release = ref<ReleaseDetail>(emptyReleaseDetail())
const releaseId = computed(() => String(route.params.id || release.value.id))
const routeAgentTaskId = computed(() => (route.query.agentTaskId ? String(route.query.agentTaskId) : ''))
const agentTaskId = computed(() => release.value.agentTaskId || routeAgentTaskId.value)
const logTitleId = computed(() => agentTaskId.value || release.value.id)
const runtimeState = computed(() => agentStatus.value?.status?.status ?? '')
const runtimeStep = computed(() => agentStatus.value?.status?.step?.trim() ?? '')
const statusToneMap: Record<string, 'good' | 'warn' | 'bad'> = {
  SUCCESS: 'good',
  COMPLETED: 'good',
  RUNNING: 'warn',
  BUILDING: 'warn',
  JENKINS_TRIGGERING: 'warn',
  JENKINS_QUEUED: 'warn',
  QUEUED: 'warn',
  PENDING: 'warn',
  PENDING_IMAGE_SYNC: 'warn',
  PENDING_CONFIRM: 'warn',
  WAITING_CONFIRM: 'warn',
  FAILED: 'bad',
  FAILURE: 'bad',
  PARTIAL_FAILED: 'bad',
  CANCELLED: 'bad',
  ABORTED: 'bad',
  UNSTABLE: 'bad',
  NOT_BUILT: 'bad',
}
const runtimeLabelMap: Record<string, string> = {
  PENDING: '待执行',
  PENDING_CONFIRM: '待确认',
  RUNNING: '执行中',
  BUILDING: '构建中',
  JENKINS_TRIGGERING: 'Jenkins 触发中',
  JENKINS_QUEUED: 'Jenkins 排队',
  QUEUED: '排队',
  PENDING_IMAGE_SYNC: '等待镜像同步',
  WAITING_CONFIRM: '等待确认',
  SUCCESS: '执行成功',
  COMPLETED: '执行完成',
  FAILED: '执行失败',
  FAILURE: '构建失败',
  PARTIAL_FAILED: '部分失败',
  CANCELLED: '已取消',
  ABORTED: '已中止',
  UNSTABLE: '不稳定',
  NOT_BUILT: '未构建',
}
const terminalStatuses = ['SUCCESS', 'FAILED', 'PARTIAL_FAILED', 'CANCELLED', 'FAILURE', 'ABORTED', 'UNSTABLE', 'NOT_BUILT', 'TRIGGER_FAILED']
const postSuccessPollLimit = 40
const postSuccessPollCount = ref(0)
const agentLogs = computed(() => {
  const logs = agentStatus.value?.logs ?? []
  return logs.length > 0 ? logs : release.value.logs ?? []
})
const isJenkinsRelease = computed(() => release.value.releaseSource === 'JENKINS_JOB')
const displayLogs = computed(() => (isJenkinsRelease.value ? release.value.logs ?? [] : agentLogs.value))
const logPanelTitle = computed(() => (isJenkinsRelease.value ? 'Jenkins 构建日志' : 'Agent 执行日志'))
const logTerminalTitle = computed(() => {
  if (isJenkinsRelease.value) return `${release.value.jenkinsJobName || 'Jenkins Pipeline'} / ${release.value.buildId || release.value.id}`
  return `${release.value.agentName} / ${logTitleId.value}`
})
const agentBadge = computed(() => agentStatus.value?.status?.status ?? (agentStatus.value?.enabled === false ? 'disabled' : 'live'))
const logBadge = computed(() => (isJenkinsRelease.value ? release.value.buildStatus || release.value.status || 'Jenkins' : agentBadge.value))
const actionRecords = computed(() => release.value.actionRecords ?? [])
const affectedServiceLabel = computed(() => {
  const services = release.value.serviceNames?.length ? release.value.serviceNames : release.value.failures.map((item) => item.serviceName)
  return services.length > 0 ? services.join('、') : '未记录'
})
const releaseSourceLabel = computed(() => {
  if (release.value.releaseSource === 'LOCAL_HARBOR_IMAGE') return '本地 Harbor 镜像'
  if (release.value.releaseSource === 'JENKINS_JOB') return 'Jenkins Job'
  return '服务'
})
const releaseImageLabel = computed(() => {
  if (release.value.imageRepository && release.value.imageTag) return `${release.value.imageRepository}:${release.value.imageTag}`
  return release.value.imageRepository || release.value.imageTag || '等待环境确认'
})
const releaseSourceSummary = computed(() => {
  if (release.value.releaseSource === 'LOCAL_HARBOR_IMAGE') return `选择镜像 ${releaseImageLabel.value}`
  if (release.value.releaseSource === 'JENKINS_JOB') return `Jenkins ${release.value.buildStatus || '触发中'}`
  return '由项目 Agent 执行镜像同步与 tag 更新'
})
const executionModeLabel = computed(() => {
  const type = agentStatus.value?.status?.type
  if (type === 'release') return '实时 Agent'
  return agentStatus.value?.enabled === false ? '静态快照' : agentTaskId.value ? 'Agent 编排' : '静态回放'
})
const effectiveReleaseStatus = computed(() => runtimeState.value || release.value.status)
const releaseStatusLabel = computed(() => runtimeLabelMap[effectiveReleaseStatus.value] ?? (effectiveReleaseStatus.value || '-'))
const releaseStatusTone = computed(() => statusToneMap[effectiveReleaseStatus.value] ?? 'warn')
const statusFoot = computed(() => runtimeStep.value || releaseSourceSummary.value)
const executorLabel = computed(() => (isJenkinsRelease.value ? release.value.jenkinsJobName || 'Jenkins Pipeline' : release.value.agentName || executionModeLabel.value))
const executorFoot = computed(() => release.value.buildId || agentTaskId.value || release.value.buildUrl || '待生成执行任务')
const executorTone = computed(() => (release.value.buildId || agentTaskId.value ? 'good' : 'warn'))
const imageFoot = computed(() => {
  if (release.value.imageDigest) return release.value.imageDigest
  if (release.value.imageRepository && release.value.imageTag) return '发版目标镜像'
  return 'Jenkins 发版后以实际 K8s/Agent 上报为准'
})
const jenkinsRuntimeStatus = computed(() => (release.value.buildStatus || release.value.status || '').trim().toUpperCase())
const shouldPollJenkinsRelease = computed(() => {
  if (!isJenkinsRelease.value) return false
  const status = jenkinsRuntimeStatus.value
  if (!terminalStatuses.includes(status)) return true
  return status === 'SUCCESS' && postSuccessPollCount.value < postSuccessPollLimit
})
let pollingTimer: number | undefined

async function loadRelease(options: { silent?: boolean } = {}) {
  if (!options.silent) {
    loading.value = true
  }
  try {
    release.value = await getReleaseDetail(releaseId.value)
  } catch (error) {
    if (!options.silent) {
      ElMessage.error(error instanceof Error ? error.message : '加载发布详情失败')
    }
  } finally {
    if (!options.silent) {
      loading.value = false
    }
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

async function pollRuntimeState() {
  if (shouldPollJenkinsRelease.value) {
    await loadRelease({ silent: true })
    if (jenkinsRuntimeStatus.value === 'SUCCESS') {
      postSuccessPollCount.value += 1
    } else {
      postSuccessPollCount.value = 0
    }
    return
  }
  await pollAgentStatus()
}

onMounted(async () => {
  await loadRelease()
  await pollAgentStatus()
  pollingTimer = window.setInterval(pollRuntimeState, 3000)
})

watch(
  () => route.fullPath,
  async () => {
    if (pollingTimer) {
      window.clearInterval(pollingTimer)
    }
    postSuccessPollCount.value = 0
    await loadRelease()
    await pollAgentStatus()
    pollingTimer = window.setInterval(pollRuntimeState, 3000)
  },
)

onUnmounted(() => {
  if (pollingTimer) {
    window.clearInterval(pollingTimer)
  }
})
</script>

<style scoped>
.release-brief {
  align-items: center;
  background: #fbfcfe;
  border: 1px solid #edf1f6;
  border-radius: 6px;
  color: #606a7b;
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-bottom: 12px;
  padding: 10px 12px;
}

.release-brief span,
.release-brief a {
  font-size: 13px;
  overflow-wrap: anywhere;
}

.release-log-scroll {
  height: clamp(360px, calc(100vh - 360px), 620px);
  overflow: hidden;
  border-radius: 8px;
}

.release-log-scroll :deep(.terminal) {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.release-log-scroll :deep(.terminal-body) {
  flex: 1;
  max-height: none;
  min-height: 0;
  overflow: auto;
}

@media (max-width: 900px) {
  .release-brief {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
