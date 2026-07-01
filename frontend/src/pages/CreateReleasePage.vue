<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>创建发布单</h1>
        <p>强化批量服务选择、风险确认、自动回滚与跳过异常 workload 策略。</p>
      </div>
      <el-button type="primary" :loading="submitting" :disabled="submitDisabled" @click="submitRelease">{{ submitText }}</el-button>
    </div>

    <div class="two-col">
      <el-card shadow="never">
        <template #header><div class="panel-head"><strong>{{ configTitle }}</strong><el-tag>{{ configTag }}</el-tag></div></template>
        <el-form label-position="top" class="form-grid">
          <el-form-item label="发版方式">
            <el-radio-group v-model="releaseMode">
              <el-radio-button label="SERVICE_RELEASE">服务发版</el-radio-button>
              <el-radio-button label="SERVICE_DEPLOYMENT">服务部署</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="目标环境">
            <el-select v-model="targetEnvironmentId" placeholder="选择目标环境">
              <el-option
                v-for="environment in environments"
                :key="environment.id"
                :label="`${environment.name} / ${environment.code}`"
                :value="environment.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE'" label="发版来源">
            <el-radio-group v-model="releaseSource">
              <el-radio-button label="JENKINS_JOB">Jenkins Job</el-radio-button>
              <el-radio-button label="LOCAL_HARBOR_IMAGE">本地 Harbor 镜像</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE' && releaseSource === 'JENKINS_JOB'" label="Jenkins job">
            <el-select v-model="jenkinsJob" :disabled="jenkinsJobOptions.length === 0">
              <el-option
                v-for="job in jenkinsJobOptions"
                :key="job"
                :label="job"
                :value="job"
              />
            </el-select>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE' && releaseSource === 'LOCAL_HARBOR_IMAGE'" label="本地 Harbor 镜像 tag">
            <el-select v-model="imageTag" :disabled="imageTagOptions.length === 0">
              <el-option
                v-for="tag in imageTagOptions"
                :key="tag"
                :label="`${imageRepository} / ${tag}`"
                :value="tag"
              />
            </el-select>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_DEPLOYMENT'" label="来源基线 / 生产环境">
            <el-select v-model="sourceBaselineId" placeholder="选择来源基线" disabled>
              <el-option :label="baselineOptionLabel" :value="sourceBaselineId" />
            </el-select>
          </el-form-item>
          <el-form-item label="执行 Agent">
            <el-select v-model="agentId" placeholder="选择执行 Agent" :disabled="availableAgents.length === 0">
              <el-option
                v-for="agent in availableAgents"
                :key="agent.id"
                :label="`${agent.name} / ${agent.status === 'ONLINE' ? '在线' : '离线'}`"
                :value="agent.id"
              />
            </el-select>
          </el-form-item>
        </el-form>
      </el-card>

      <ReleaseRiskPanel v-model:options="options" :selected-count="selectedIds.length" />
    </div>

    <el-alert
      class="readiness-alert"
      :type="readinessAlertType"
      :closable="false"
      :title="readinessTitle"
      :description="readinessDescription"
      show-icon
    />

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <strong>{{ selectionTitle }} <span class="mono">{{ selectedIds.length }}</span> 个服务</strong>
          <el-input v-model="keyword" placeholder="搜索服务、namespace、tag" clearable />
        </div>
        <div class="top-actions">
          <el-button @click="selectPublishable">{{ selectAllText }}</el-button>
          <el-button @click="selectedIds = []">清空选择</el-button>
        </div>
      </div>
      <ServiceDiffTable v-model:selected-ids="selectedIds" :items="filteredItems" />
    </el-card>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ReleaseRiskPanel from '@/components/ReleaseRiskPanel.vue'
import ServiceDiffTable from '@/components/ServiceDiffTable.vue'
import { listAgents } from '@/api/agents'
import { createDeployTask } from '@/api/deployTasks'
import { getBaselineCompare, getBaselineDetail, type BaselineDiffItem, type BaselineDiffResult } from '@/api/baselines'
import { listEnvironments } from '@/api/environments'
import { createRelease, listReleaseSources, type ReleaseSource, type ReleaseSourceService } from '@/api/releases'
import type { AgentInfo } from '@/api/agents'
import type { EnvironmentInfo } from '@/api/environments'
import { useAuthStore } from '@/stores/authStore'
import { resolveCreateReleaseErrorMessage } from './createReleaseErrors'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const keyword = ref('')
const releaseMode = ref<'SERVICE_RELEASE' | 'SERVICE_DEPLOYMENT'>('SERVICE_RELEASE')
const releaseSource = ref<'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE'>('LOCAL_HARBOR_IMAGE')
const jenkinsJob = ref('')
const imageTag = ref('')
const selectedIds = ref<string[]>([])
const submitting = ref(false)
const baselineDetail = ref({
  id: '',
  name: '',
  sourceEnvironmentName: '',
})
const diffResult = ref({
  sourceBaselineId: '',
  targetEnvironmentId: '',
  items: [] as ServiceTableItem[],
})
const releaseSourceData = ref<ReleaseSource | null>(null)
const releaseSourceLoading = ref(false)
const releaseSourceError = ref('')
const agentsError = ref('')
const environmentsError = ref('')
const deploymentSourceError = ref('')
const agents = ref<AgentInfo[]>([])
const environments = ref<EnvironmentInfo[]>([])
const sourceBaselineId = ref('')
const targetEnvironmentId = ref('')
const agentId = ref('')
const options = ref({
  autoRollback: true,
  skipWorkloadError: true,
  refreshTargetRuntime: true,
  auditLog: true,
})

type ServiceTableItem = {
  serviceId: string
  serviceName: string
  namespace: string
  sourceTag: string
  targetTag: string | null
  diffStatus: string
  publishable: boolean
  strategy: string
}

function toServiceTableItem(item: BaselineDiffItem): ServiceTableItem {
  return {
    serviceId: item.serviceId,
    serviceName: item.serviceName,
    namespace: item.namespace,
    sourceTag: item.sourceTag ?? '',
    targetTag: item.targetTag ?? null,
    diffStatus: item.diffStatus,
    publishable: item.publishable,
    strategy: typeof item.strategy === 'string' ? item.strategy : '',
  }
}

function toDiffResult(result: BaselineDiffResult): { sourceBaselineId: string; targetEnvironmentId: string; items: ServiceTableItem[] } {
  return {
    sourceBaselineId: result.sourceBaselineId ?? result.baselineId,
    targetEnvironmentId: result.targetEnvironmentId,
    items: result.items.map(toServiceTableItem),
  }
}

const releaseSourceItems = computed<ServiceTableItem[]>(() => {
  return (releaseSourceData.value?.services ?? []).map((service) => {
    const tag = service.tags[0]
    return {
      serviceId: service.serviceId,
      serviceName: service.serviceName,
      namespace: service.namespace,
      sourceTag: tag?.tag ?? '',
      targetTag: null,
      diffStatus: service.publishable ? 'IMAGE_AVAILABLE' : 'IMAGE_UNAVAILABLE',
      publishable: service.publishable,
      strategy: service.publishable ? '选择 Harbor 镜像 tag' : service.message || 'Harbor 镜像不可用',
    }
  })
})

const candidateItems = computed(() => {
  if (releaseMode.value === 'SERVICE_RELEASE') {
    return releaseSourceItems.value
  }
  return diffResult.value.items.filter((item) =>
    item.diffStatus === 'MISSING_IN_TARGET',
  )
})

const filteredItems = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return candidateItems.value
  return candidateItems.value.filter((item) =>
    `${item.serviceName} ${item.namespace} ${item.sourceTag} ${item.targetTag ?? ''}`.toLowerCase().includes(q),
  )
})
const configTitle = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '服务部署配置' : '服务发版配置'))
const configTag = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '目标缺失服务' : '目标已有服务'))
const selectionTitle = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '待部署' : '待发版'))
const selectAllText = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '选择全部待部署' : '选择全部可发版'))
const submitText = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? '创建服务部署任务' : '提交服务发版'))
const submitDisabled = computed(() => submitting.value || readinessBlockingReasons.value.length > 0)
const baselineOptionLabel = computed(() => `${sourceBaselineId.value} / ${baselineDetail.value.sourceEnvironmentName} / ${baselineDetail.value.name}`)
const selectedEnvironment = computed(() => environments.value.find((item) => item.id === targetEnvironmentId.value))
const agentsInTargetEnvironment = computed(() => agents.value.filter((item) => item.environmentId === targetEnvironmentId.value))
const availableAgents = computed(() =>
  agentsInTargetEnvironment.value.filter((item) => item.status === 'ONLINE'),
)
const offlineAgentsInTargetEnvironment = computed(() =>
  agentsInTargetEnvironment.value.filter((item) => item.status !== 'ONLINE'),
)
const requiredPermission = computed(() => (releaseMode.value === 'SERVICE_DEPLOYMENT' ? 'deploy:write' : 'release:write'))
const missingPermission = computed(() => !authStore.hasPermission(requiredPermission.value))
const readinessBlockingReasons = computed(() => {
  const reasons: string[] = []
  if (!targetEnvironmentId.value || !selectedEnvironment.value) {
    reasons.push('请选择有效的目标环境')
  }
  if (environmentsError.value) {
    reasons.push(environmentsError.value)
  }
  if (agentsError.value) {
    reasons.push(agentsError.value)
  }
  if (releaseMode.value === 'SERVICE_DEPLOYMENT' && deploymentSourceError.value) {
    reasons.push(deploymentSourceError.value)
  }
  if (missingPermission.value) {
    reasons.push(releaseMode.value === 'SERVICE_DEPLOYMENT' ? '当前账号没有服务部署权限' : '当前账号没有服务发版权限')
  }
  if (targetEnvironmentId.value && availableAgents.value.length === 0) {
    const offlineCount = offlineAgentsInTargetEnvironment.value.length
    reasons.push(offlineCount > 0 ? `目标环境有 ${offlineCount} 个 Agent，但当前都不在线` : '目标环境尚未接入在线 Agent')
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSourceLoading.value) {
    reasons.push('正在读取 Harbor 发布源')
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSourceError.value) {
    reasons.push(releaseSourceError.value)
  }
  if (selectedIds.value.length === 0) {
    reasons.push(releaseMode.value === 'SERVICE_DEPLOYMENT' ? '请选择目标环境缺失服务' : '请选择目标已有且需要更新的服务')
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSource.value === 'LOCAL_HARBOR_IMAGE' && selectedIds.value.length > 1) {
    reasons.push('本地 Harbor 镜像发版当前一次只能选择一个服务')
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSource.value === 'LOCAL_HARBOR_IMAGE' && (!imageRepository.value || !imageTag.value)) {
    reasons.push('请选择可用的 Harbor 镜像 tag')
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSource.value === 'JENKINS_JOB' && !jenkinsJob.value) {
    reasons.push('当前环境未返回可用 Jenkins job')
  }
  return reasons
})
const readinessAlertType = computed(() => (readinessBlockingReasons.value.length > 0 ? 'warning' : 'success'))
const readinessTitle = computed(() => (readinessBlockingReasons.value.length > 0 ? '创建前检查未通过' : '创建前检查已通过'))
const readinessDescription = computed(() => {
  if (readinessBlockingReasons.value.length > 0) {
    return readinessBlockingReasons.value.join('；')
  }
  const environmentName = selectedEnvironment.value?.name || targetEnvironmentId.value
  const agentName = availableAgents.value.find((item) => item.id === agentId.value)?.name || agentId.value
  return `${environmentName} 已选择在线 Agent ${agentName}，可创建${releaseMode.value === 'SERVICE_DEPLOYMENT' ? '服务部署任务' : '服务发版任务'}`
})
const releaseServiceById = computed(() => {
  const services = new Map<string, ReleaseSourceService>()
  ;(releaseSourceData.value?.services ?? []).forEach((service) => services.set(service.serviceId, service))
  return services
})
const selectedReleaseService = computed(() => releaseServiceById.value.get(selectedIds.value[0]))
const jenkinsJobOptions = computed(() => releaseSourceData.value?.jenkinsJobs ?? [])
const imageTagOptions = computed(() => {
  return (selectedReleaseService.value?.tags ?? []).map((item) => item.tag).filter(Boolean)
})
const selectedImageTag = computed(() => selectedReleaseService.value?.tags.find((item) => item.tag === imageTag.value))
const imageRepository = computed(() => selectedReleaseService.value?.imageRepository ?? '')

function selectPublishable() {
  selectedIds.value = filteredItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
}

function syncSelectedIds() {
  const candidateIds = new Set(candidateItems.value.map((item) => item.serviceId))
  selectedIds.value = selectedIds.value.filter((id) => candidateIds.has(id))
}

function syncReleaseSourceFields() {
  if (!jenkinsJobOptions.value.includes(jenkinsJob.value)) {
    jenkinsJob.value = jenkinsJobOptions.value[0] || ''
  }
  if (!imageTagOptions.value.includes(imageTag.value)) {
    imageTag.value = imageTagOptions.value[0] || ''
  }
}

function syncAgentId() {
  if (availableAgents.value.some((item) => item.id === agentId.value)) return
  agentId.value = availableAgents.value[0]?.id || ''
}

function syncTargetEnvironmentId() {
  const routeEnvironmentId = String(route.query.targetEnvironmentId || '')
  const currentEnvironmentExists = environments.value.some((item) => item.id === targetEnvironmentId.value)
  if (currentEnvironmentExists && !routeEnvironmentId) {
    syncAgentId()
    return
  }
  targetEnvironmentId.value = routeEnvironmentId || environments.value[0]?.id || ''
  syncAgentId()
}

watch(releaseMode, () => {
  keyword.value = ''
  syncSelectedIds()
  syncReleaseSourceFields()
}, { immediate: true })

async function loadReleaseSources() {
  if (releaseMode.value !== 'SERVICE_RELEASE' || !targetEnvironmentId.value) {
    releaseSourceData.value = null
    releaseSourceError.value = ''
    syncSelectedIds()
    syncReleaseSourceFields()
    return
  }
  releaseSourceLoading.value = true
  releaseSourceError.value = ''
  try {
    releaseSourceData.value = await listReleaseSources(targetEnvironmentId.value, keyword.value)
  } catch {
    releaseSourceData.value = null
    releaseSourceError.value = '读取 Harbor 发布源失败'
  } finally {
    releaseSourceLoading.value = false
    syncSelectedIds()
    if (selectedIds.value.length === 0) {
      selectPublishable()
    }
    syncReleaseSourceFields()
  }
}

async function submitRelease() {
  if (missingPermission.value) {
    ElMessage.warning(releaseMode.value === 'SERVICE_DEPLOYMENT' ? '当前账号没有服务部署权限' : '当前账号没有服务发版权限')
    return
  }
  if (!targetEnvironmentId.value) {
    ElMessage.warning('请选择目标环境')
    return
  }
  if (!agentId.value) {
    ElMessage.warning('当前目标环境没有可用的在线 Agent')
    return
  }
  if (selectedIds.value.length === 0) {
    ElMessage.warning(releaseMode.value === 'SERVICE_DEPLOYMENT' ? '请至少选择一个待部署服务' : '请至少选择一个待发版服务')
    return
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSource.value === 'LOCAL_HARBOR_IMAGE') {
    if (selectedIds.value.length > 1) {
      ElMessage.warning('本地 Harbor 镜像发版当前一次只能选择一个服务')
      return
    }
    if (!selectedReleaseService.value || !imageRepository.value || !imageTag.value) {
      ElMessage.warning('请选择可用的 Harbor 镜像 tag')
      return
    }
  }
  if (releaseMode.value === 'SERVICE_RELEASE' && releaseSource.value === 'JENKINS_JOB' && !jenkinsJob.value) {
    ElMessage.warning('当前环境未返回可用 Jenkins job')
    return
  }
  submitting.value = true
  try {
    if (releaseMode.value === 'SERVICE_DEPLOYMENT') {
      const result = await createDeployTask({
        type: 'SERVICE_DEPLOYMENT',
        sourceBaselineId: diffResult.value.sourceBaselineId,
        targetEnvironmentId: targetEnvironmentId.value,
        agentId: agentId.value,
        serviceIds: selectedIds.value,
        options: {
          syncImage: true,
          createWorkload: true,
          healthCheck: true,
        },
      })
      ElMessage.success('服务部署任务已创建')
      router.push({
        path: `/deploy-tasks/${result.id}`,
        query: result.agentTaskId ? { agentTaskId: result.agentTaskId } : undefined,
      })
      return
    }

    const result = await createRelease({
      type: 'SERVICE_RELEASE',
      targetEnvironmentId: targetEnvironmentId.value,
      agentId: agentId.value,
      serviceIds: selectedIds.value,
      releaseSource: releaseSource.value,
      image: releaseSource.value === 'LOCAL_HARBOR_IMAGE' ? {
        repository: imageRepository.value,
        tag: imageTag.value,
        digest: selectedImageTag.value?.digest,
      } : undefined,
      jenkins: releaseSource.value === 'JENKINS_JOB' ? {
        jobName: jenkinsJob.value,
        branch: '',
        parameters: {
          SERVICE_COUNT: String(selectedIds.value.length),
          TARGET_ENV: targetEnvironmentId.value,
          RELEASE_SOURCE: releaseSource.value,
          IMAGE_TAG: imageTag.value,
        },
      } : undefined,
      options: options.value,
    })
    ElMessage.success(releaseSource.value === 'LOCAL_HARBOR_IMAGE' ? '服务发版已提交镜像同步' : '服务发版已提交 Jenkins')
    router.push({
      path: `/releases/${result.id}`,
      query: result.agentTaskId ? { agentTaskId: result.agentTaskId } : undefined,
    })
  } catch (error) {
    ElMessage.error(resolveCreateReleaseErrorMessage(error, releaseMode.value))
  } finally {
    submitting.value = false
  }
}

async function loadAgents() {
  agentsError.value = ''
  try {
    agents.value = await listAgents()
  } catch {
    agents.value = []
    agentsError.value = '读取 Agent 列表失败'
  } finally {
    syncAgentId()
  }
}

async function loadEnvironments() {
  environmentsError.value = ''
  try {
    environments.value = await listEnvironments()
  } catch {
    environments.value = []
    environmentsError.value = '读取环境列表失败'
  } finally {
    syncTargetEnvironmentId()
    void loadReleaseSources()
  }
}

async function loadDiffResult() {
  releaseMode.value = route.query.mode === 'SERVICE_DEPLOYMENT' ? 'SERVICE_DEPLOYMENT' : 'SERVICE_RELEASE'
  selectedIds.value = route.query.serviceIds ? String(route.query.serviceIds).split(',').filter(Boolean) : []
  deploymentSourceError.value = ''
  if (releaseMode.value === 'SERVICE_RELEASE') {
    baselineDetail.value = { id: '', name: '', sourceEnvironmentName: '' }
    diffResult.value = { sourceBaselineId: '', targetEnvironmentId: '', items: [] }
    sourceBaselineId.value = ''
    syncTargetEnvironmentId()
    syncSelectedIds()
    await loadReleaseSources()
    return
  }
  const baselineId = String(route.query.baselineId || diffResult.value.sourceBaselineId || '')
  if (!baselineId) {
    deploymentSourceError.value = '服务部署需要先选择来源基线'
    sourceBaselineId.value = ''
    diffResult.value = { sourceBaselineId: '', targetEnvironmentId: targetEnvironmentId.value, items: [] }
    syncTargetEnvironmentId()
    syncSelectedIds()
    return
  }
  const routeTargetEnvironmentId = String(route.query.targetEnvironmentId || '')
  sourceBaselineId.value = baselineId
  try {
    const [detail, result] = await Promise.all([
      getBaselineDetail(baselineId),
      getBaselineCompare(baselineId, routeTargetEnvironmentId),
    ])
    baselineDetail.value = detail
    diffResult.value = toDiffResult(result)
    sourceBaselineId.value = diffResult.value.sourceBaselineId
    syncTargetEnvironmentId()
    syncSelectedIds()
    if (selectedIds.value.length === 0) {
      selectedIds.value = candidateItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
    }
  } catch {
    baselineDetail.value = { id: '', name: '', sourceEnvironmentName: '' }
    diffResult.value = { sourceBaselineId: baselineId, targetEnvironmentId: routeTargetEnvironmentId || targetEnvironmentId.value, items: [] }
    deploymentSourceError.value = '读取基线差异失败'
    syncTargetEnvironmentId()
    syncSelectedIds()
  }
}

watch(targetEnvironmentId, () => {
  syncAgentId()
  void loadReleaseSources()
})
watch(releaseSource, syncReleaseSourceFields)
watch(selectedIds, syncReleaseSourceFields, { immediate: true })
watch(keyword, () => {
  void loadReleaseSources()
})

loadAgents()
loadEnvironments()
watch(() => route.fullPath, loadDiffResult, { immediate: true })
</script>
