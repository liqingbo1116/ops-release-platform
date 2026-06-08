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
            <el-select v-model="jenkinsJob">
              <el-option
                v-for="job in jenkinsJobOptions"
                :key="job"
                :label="job"
                :value="job"
              />
            </el-select>
          </el-form-item>
          <el-form-item v-if="releaseMode === 'SERVICE_RELEASE' && releaseSource === 'LOCAL_HARBOR_IMAGE'" label="本地 Harbor 镜像 tag">
            <el-select v-model="imageTag">
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
import { getBaselineCompare, getBaselineDetail } from '@/api/baselines'
import { listEnvironments } from '@/api/environments'
import { createRelease } from '@/api/releases'
import { agentMockData } from '@/api/mockData/agent'
import { baselineMockData } from '@/api/mockData/baseline'
import { environmentMockData } from '@/api/mockData/environment'
import { resolveCreateReleaseErrorMessage } from './createReleaseErrors'

const route = useRoute()
const router = useRouter()
const keyword = ref('')
const releaseMode = ref<'SERVICE_RELEASE' | 'SERVICE_DEPLOYMENT'>('SERVICE_RELEASE')
const releaseSource = ref<'JENKINS_JOB' | 'LOCAL_HARBOR_IMAGE'>('JENKINS_JOB')
const jenkinsJob = ref('')
const imageTag = ref('')
const selectedIds = ref<string[]>([])
const submitting = ref(false)
const baselineDetail = ref({ ...baselineMockData.baselineDetail })
const diffResult = ref({ ...baselineMockData.diffResult })
const agents = ref<typeof agentMockData.agents>([])
const environments = ref<typeof environmentMockData.environments>([])
const sourceBaselineId = ref('')
const targetEnvironmentId = ref('')
const agentId = ref('')
const options = ref({
  autoRollback: true,
  skipWorkloadError: true,
  refreshTargetRuntime: true,
  auditLog: true,
})

const candidateItems = computed(() => {
  return diffResult.value.items.filter((item) =>
    releaseMode.value === 'SERVICE_DEPLOYMENT'
      ? item.diffStatus === 'MISSING_IN_TARGET'
      : item.diffStatus === 'NEED_UPDATE',
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
const submitDisabled = computed(() => submitting.value || selectedIds.value.length === 0 || !targetEnvironmentId.value || !agentId.value)
const baselineOptionLabel = computed(() => `${sourceBaselineId.value} / ${baselineDetail.value.sourceEnvironmentName} / ${baselineDetail.value.name}`)
const availableAgents = computed(() =>
  agents.value.filter((item) => item.environmentId === targetEnvironmentId.value && item.status === 'ONLINE'),
)
const selectedServices = computed(() => {
  const selectedSet = new Set(selectedIds.value)
  return candidateItems.value.filter((item) => selectedSet.has(item.serviceId))
})
const releaseServices = computed(() => selectedServices.value.filter((item) => item.diffStatus === 'NEED_UPDATE'))
const serviceSlug = computed(() => {
  const firstService = releaseServices.value[0]?.serviceName || candidateItems.value[0]?.serviceName || 'service'
  return firstService.replace(/-service$/, '').replace(/[^a-z0-9]+/gi, '-').replace(/^-+|-+$/g, '').toLowerCase()
})
const environmentCode = computed(() => {
  const currentEnvironment = environments.value.find((item) => item.id === targetEnvironmentId.value)
  return currentEnvironment?.code || 'target-env'
})
const projectSlug = computed(() => {
  const parts = environmentCode.value.split('-')
  return parts.slice(0, Math.max(parts.length - 1, 1)).join('-')
})
const jenkinsJobOptions = computed(() => [
  `${projectSlug.value}-${serviceSlug.value}-release`,
  `${projectSlug.value}-image-tag-release`,
])
const imageTagOptions = computed(() => {
  const tags = releaseServices.value.map((item) => item.sourceTag)
  return [...new Set(tags.length > 0 ? tags : candidateItems.value.map((item) => item.sourceTag))].filter(Boolean)
})
const imageRepository = computed(() => `harbor.local/${projectSlug.value}/${serviceSlug.value || 'service'}`)

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
  const diffEnvironmentId = diffResult.value.targetEnvironmentId
  targetEnvironmentId.value = routeEnvironmentId || diffEnvironmentId || environments.value[0]?.id || ''
  syncAgentId()
}

watch(releaseMode, () => {
  keyword.value = ''
  syncSelectedIds()
  selectPublishable()
  syncReleaseSourceFields()
}, { immediate: true })

async function submitRelease() {
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
      sourceBaselineId: diffResult.value.sourceBaselineId,
      targetEnvironmentId: targetEnvironmentId.value,
      agentId: agentId.value,
      serviceIds: selectedIds.value,
      releaseSource: releaseSource.value,
      image: releaseSource.value === 'LOCAL_HARBOR_IMAGE' ? {
        repository: imageRepository.value,
        tag: imageTag.value,
        digest: `sha256:mock-${imageTag.value}`,
      } : undefined,
      jenkins: releaseSource.value === 'JENKINS_JOB' ? {
        jobName: jenkinsJob.value,
        branch: 'main',
        parameters: {
          SERVICE_COUNT: String(selectedIds.value.length),
          TARGET_ENV: targetEnvironmentId.value,
          RELEASE_SOURCE: releaseSource.value,
          IMAGE_TAG: imageTag.value,
        },
      } : undefined,
      options: options.value,
    })
    ElMessage.success('服务发版已提交 Jenkins')
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
  try {
    agents.value = await listAgents()
  } catch {
    agents.value = [...agentMockData.agents]
  } finally {
    syncAgentId()
  }
}

async function loadEnvironments() {
  try {
    environments.value = await listEnvironments()
  } catch {
    environments.value = [...environmentMockData.environments]
  } finally {
    syncTargetEnvironmentId()
  }
}

async function loadDiffResult() {
  releaseMode.value = route.query.mode === 'SERVICE_DEPLOYMENT' ? 'SERVICE_DEPLOYMENT' : 'SERVICE_RELEASE'
  selectedIds.value = route.query.serviceIds ? String(route.query.serviceIds).split(',').filter(Boolean) : []
  const baselineId = String(route.query.baselineId || diffResult.value.sourceBaselineId || 'BL-20260607-0001')
  const routeTargetEnvironmentId = String(route.query.targetEnvironmentId || '')
  sourceBaselineId.value = baselineId
  try {
    const [detail, result] = await Promise.all([
      getBaselineDetail(baselineId),
      getBaselineCompare(baselineId, routeTargetEnvironmentId),
    ])
    baselineDetail.value = detail
    diffResult.value = result
    sourceBaselineId.value = result.sourceBaselineId
    syncTargetEnvironmentId()
    syncSelectedIds()
    if (selectedIds.value.length === 0) {
      selectedIds.value = candidateItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
    }
  } catch {
    baselineDetail.value = { ...baselineMockData.baselineDetail }
    diffResult.value = { ...baselineMockData.diffResult }
    sourceBaselineId.value = diffResult.value.sourceBaselineId
    syncTargetEnvironmentId()
    syncSelectedIds()
    if (selectedIds.value.length === 0) {
      selectedIds.value = candidateItems.value.filter((item) => item.publishable).map((item) => item.serviceId)
    }
  }
}

watch(targetEnvironmentId, syncAgentId)
watch([selectedIds, targetEnvironmentId], syncReleaseSourceFields, { immediate: true })

loadAgents()
loadEnvironments()
watch(() => route.fullPath, loadDiffResult, { immediate: true })
</script>
