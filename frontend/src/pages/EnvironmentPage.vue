<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境管理</h1>
        <p>本地环境由平台直连基础资源；远程环境由 Agent 上报状态并执行任务。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadAll">刷新</el-button>
        <el-button type="primary" @click="openCreateDialog">新增环境</el-button>
      </div>
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 主线：本地环境关联 K8s 命名空间、Harbor 镜像项目、Jenkins 流水线视图；远程环境关联本地 Harbor 镜像项目与 Jenkins 流水线视图，远程 K8s 由 Agent 执行。"
      />
      <el-alert
        type="info"
        :closable="false"
        title="这些资源范围用于后续服务关联：服务会在环境内选择实际使用的命名空间、镜像项目和流水线视图，发布/部署时据此确定构建来源、镜像范围和部署目标。"
      />
      <el-alert
        v-if="blockedProjectEnvironmentCount > 0"
        type="warning"
        :closable="false"
        :title="`${blockedProjectEnvironmentCount} 个远程环境 Agent 未就绪，远程发布/部署提交前会被阻断。`"
      />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="keyword" placeholder="搜索环境、标识" clearable />
          <el-select v-model="environmentType" placeholder="全部环境类型" clearable>
            <el-option label="本地环境" value="LOCAL" />
            <el-option label="远程环境" value="PROJECT" />
          </el-select>
        </div>
      </div>
      <el-alert v-if="errorMessage" class="environment-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="name" label="环境" min-width="150" />
        <el-table-column prop="code" label="环境标识" min-width="150" />
        <el-table-column label="类型" min-width="110">
          <template #default="{ row }">{{ environmentTypeLabel(row.type) }}</template>
        </el-table-column>
        <el-table-column label="部署目标" min-width="110">
          <template #default="{ row }">{{ deployTargetTypeLabel(row.deployTargetType) }}</template>
        </el-table-column>
        <el-table-column label="K8s / 命名空间" min-width="210">
          <template #default="{ row }">{{ scopedResourceText(row, 'k8s') }}</template>
        </el-table-column>
        <el-table-column label="Harbor / 镜像项目" min-width="210">
          <template #default="{ row }">{{ scopedResourceText(row, 'harbor') }}</template>
        </el-table-column>
        <el-table-column label="Jenkins / 流水线视图" min-width="210">
          <template #default="{ row }">{{ scopedResourceText(row, 'jenkins') }}</template>
        </el-table-column>
        <el-table-column label="Agent" min-width="100">
          <template #default="{ row }">
            <span v-if="row.type === 'LOCAL'">无需 Agent</span>
            <StatusTag v-else :status="row.agentStatus" />
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="问题 / 下一步" min-width="320">
          <template #default="{ row }">
            <div class="diagnosis-cell">
              <strong>{{ diagnosisSummary(row).message }}</strong>
              <span>{{ diagnosisSummary(row).nextStep }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="170">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="primary" @click="openDrawer(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <EnvironmentConfigDrawer
      v-model:visible="drawerVisible"
      :environment="activeEnvironment"
      :resource-name="drawerResourceName"
      :checking="checkingEnvironment"
      :diagnostics="activeDiagnostics"
      :check-help-text="activeCheckHelpText"
      @check="handleCheckEnvironment"
    />

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增环境' : '编辑环境'" width="580px" destroy-on-close>
      <el-form :model="form" label-width="120px">
        <el-form-item label="环境名称" required><el-input v-model="form.name" placeholder="项目 X 生产" /></el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" label="环境标识">
          <el-input v-model="form.code" placeholder="保存时自动生成" disabled />
          <div class="form-tip">由系统自动生成，用于远程 Agent 配置；保存后系统生成环境 ID：env-环境标识</div>
        </el-form-item>
        <el-form-item label="环境类型" required><el-segmented v-model="form.type" :options="typeOptions" /></el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" label="部署目标">
          <el-input :value="deployTargetTypeLabel(form.deployTargetType)" disabled />
          <div class="form-tip">V1 当前支持 Kubernetes；docker-compose 仅预留模型，不进入当前主线实现。</div>
        </el-form-item>
        <template v-if="form.type === 'LOCAL'">
          <el-form-item label="K8s 集群" required>
            <el-select v-model="form.clusterId" placeholder="先在基础资源维护 K8s 集群" filterable>
              <el-option v-for="item in kubernetesClusters" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="命名空间" required>
            <el-select
              v-model="form.namespaces"
              placeholder="选择或输入一个或多个命名空间"
              multiple
              filterable
              allow-create
              default-first-option
            >
              <el-option v-for="item in selectedNamespaces" :key="item" :label="item" :value="item" />
            </el-select>
            <div class="form-tip">可绑定多个命名空间，首个命名空间作为默认部署目标；手工输入未探测到的值后，环境会进入需验证状态。</div>
          </el-form-item>
          <el-form-item label="Harbor 仓库" required>
            <el-select v-model="form.registryId" placeholder="先在基础资源维护 Harbor 仓库" filterable>
              <el-option v-for="item in harborRegistries" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="镜像项目" required>
            <el-select
              v-model="form.registryProjects"
              placeholder="选择或输入一个或多个镜像项目"
              multiple
              filterable
              allow-create
              default-first-option
            >
              <el-option v-for="item in selectedProjects" :key="item" :label="item" :value="item" />
            </el-select>
            <div class="form-tip">可绑定多个镜像项目，首个镜像项目作为默认值；手工输入未探测到的值后，环境会进入需验证状态。</div>
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="Harbor 仓库" required>
            <el-select v-model="form.registryId" placeholder="选择本地 Harbor 仓库" filterable>
              <el-option v-for="item in harborRegistries" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="镜像项目" required>
            <el-select
              v-model="form.registryProjects"
              placeholder="选择或输入一个或多个本地镜像项目"
              multiple
              filterable
              allow-create
              default-first-option
            >
              <el-option v-for="item in selectedProjects" :key="item" :label="item" :value="item" />
            </el-select>
            <div class="form-tip">远程环境可绑定多个本地镜像项目，首个镜像项目作为默认值；手工输入未探测到的值后，环境会进入需验证状态。</div>
          </el-form-item>
        </template>
        <el-form-item label="Jenkins">
          <el-select v-model="form.jenkinsId" placeholder="选择 Jenkins 资源" clearable filterable>
            <el-option v-for="item in jenkinsInstances" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.jenkinsId" label="流水线视图">
          <el-select
            v-model="form.jenkinsViews"
            placeholder="选择或输入一个或多个流水线视图"
            multiple
            filterable
            allow-create
            default-first-option
          >
            <el-option v-for="item in selectedViews" :key="item" :label="item" :value="item" />
          </el-select>
          <div class="form-tip">可绑定多个流水线视图，首个流水线视图作为默认值；手工输入未探测到的值后，环境会进入需验证状态。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitEnvironment">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import EnvironmentConfigDrawer from '@/components/EnvironmentConfigDrawer.vue'
import StatusTag from '@/components/StatusTag.vue'
import {
  checkEnvironment,
  createEnvironment,
  listEnvironments,
  updateEnvironment,
  type EnvironmentCheckResult,
  type EnvironmentInfo,
  type EnvironmentPayload,
  type EnvironmentResourceBinding,
} from '@/api/environments'
import {
  listHarborRegistries,
  listJenkinsInstances,
  listKubernetesClusters,
  type HarborRegistry,
  type IntegrationResource,
  type JenkinsInstance,
  type KubernetesCluster,
} from '@/api/integrationResources'

const keyword = ref('')
const environmentType = ref('')
const drawerVisible = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const activeEnvironment = ref<EnvironmentInfo | null>(null)
const environments = ref<EnvironmentInfo[]>([])
const kubernetesClusters = ref<KubernetesCluster[]>([])
const harborRegistries = ref<HarborRegistry[]>([])
const jenkinsInstances = ref<JenkinsInstance[]>([])
const loading = ref(false)
const submitting = ref(false)
const checkingEnvironment = ref(false)
const errorMessage = ref('')
type EnvironmentDiagnostic = {
  component: string
  status: 'HEALTHY' | 'DEGRADED' | 'UNKNOWN'
  message: string
  nextStep: string
}
type EnvironmentForm = EnvironmentPayload & {
  namespaces: string[]
  registryProjects: string[]
  jenkinsViews: string[]
}
const checkResultsByEnvironmentId = ref<Record<string, EnvironmentDiagnostic[]>>({})

const form = ref<EnvironmentForm>(emptyEnvironmentForm())

const typeOptions = [
  { label: '远程环境', value: 'PROJECT' },
  { label: '本地环境', value: 'LOCAL' },
]

const blockedProjectEnvironmentCount = computed(
  () => environments.value.filter((item) => item.type === 'PROJECT' && item.agentStatus !== 'ONLINE').length,
)

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return environments.value.filter((item) => {
    const keywordMatched = !q || `${item.name} ${item.code}`.toLowerCase().includes(q)
    const typeMatched = !environmentType.value || item.type === environmentType.value
    return keywordMatched && typeMatched
  })
})

const selectedNamespaces = computed(() => kubernetesClusters.value.find((item) => item.id === form.value.clusterId)?.namespaces ?? [])
const selectedProjects = computed(() => harborRegistries.value.find((item) => item.id === form.value.registryId)?.projects ?? [])
const selectedViews = computed(() => jenkinsInstances.value.find((item) => item.id === form.value.jenkinsId)?.views ?? [])
const activeDiagnostics = computed(() => {
  if (!activeEnvironment.value) return []
  return environmentDiagnostics(activeEnvironment.value)
})
const activeCheckHelpText = computed(() => {
  if (!activeEnvironment.value) return ''
  if (activeEnvironment.value.type === 'LOCAL') {
    return '本地环境连接测试表示平台后端直接校验已绑定的 K8s 命名空间、Harbor 镜像项目和 Jenkins 流水线视图。本地环境不依赖 Agent；当前未配置真实集成时，会使用基础资源最近探测结果判断这些范围是否存在。'
  }
  return '远程环境的 K8s 操作由 Agent 执行；连接测试主要校验本地 Harbor/Jenkins 资源范围和 Agent 在线状态。'
})

function emptyEnvironmentForm(): EnvironmentForm {
  return {
    id: '',
    name: '',
    code: '',
    type: 'PROJECT',
    deployTargetType: 'KUBERNETES',
    networkMode: 'AGENT',
    clusterId: '',
    namespace: '',
    registryId: '',
    registryProject: '',
    jenkinsId: '',
    jenkinsView: '',
    bindings: [],
    namespaces: [],
    registryProjects: [],
    jenkinsViews: [],
  }
}

function resourceName(items: IntegrationResource[], id: string) {
  if (!id) return '-'
  return items.find((item) => item.id === id)?.name || id
}

function environmentTypeLabel(type: string) {
  return type === 'LOCAL' ? '本地环境' : '远程环境'
}

function deployTargetTypeLabel(type: string) {
  return type === 'DOCKER_COMPOSE' ? 'docker-compose' : 'Kubernetes'
}

function scopedResourceText(row: EnvironmentInfo, resourceType: 'k8s' | 'harbor' | 'jenkins') {
  if (row.type === 'PROJECT' && resourceType === 'k8s') return 'Agent 执行'
  const type = bindingResourceType(resourceType)
  const bindings = row.bindings?.filter((item) => item.resourceType === type) ?? []
  if (bindings.length > 0) {
    const defaultBinding = bindings.find((item) => item.isDefault) ?? bindings[0]
    const orderedBindings = [defaultBinding, ...bindings.filter((item) => item !== defaultBinding)]
    const resourceIds = Array.from(new Set(orderedBindings.map((item) => item.resourceId)))
    if (resourceIds.length === 1) {
      return `${resourceDisplayName(type, defaultBinding.resourceId)} / ${orderedBindings.map((item) => item.scopeValue).join('、')}`
    }
    return orderedBindings.map((item) => `${resourceDisplayName(type, item.resourceId)} / ${item.scopeValue}`).join('；')
  }
  if (resourceType === 'k8s') return `${resourceName(kubernetesClusters.value, row.clusterId)} / ${row.namespace || '-'}`
  if (resourceType === 'harbor') return `${resourceName(harborRegistries.value, row.registryId)} / ${row.registryProject || '-'}`
  return `${resourceName(jenkinsInstances.value, row.jenkinsId)} / ${row.jenkinsView || '-'}`
}

function bindingResourceType(resourceType: 'k8s' | 'harbor' | 'jenkins'): EnvironmentResourceBinding['resourceType'] {
  if (resourceType === 'k8s') return 'K8S'
  if (resourceType === 'harbor') return 'HARBOR'
  return 'JENKINS'
}

function resourceDisplayName(resourceType: EnvironmentResourceBinding['resourceType'], resourceId: string) {
  if (resourceType === 'K8S') return resourceName(kubernetesClusters.value, resourceId)
  if (resourceType === 'HARBOR') return resourceName(harborRegistries.value, resourceId)
  return resourceName(jenkinsInstances.value, resourceId)
}

function drawerResourceName(resourceType: EnvironmentResourceBinding['resourceType'], resourceId: string) {
  return resourceDisplayName(resourceType, resourceId)
}

function diagnosisSummary(row: EnvironmentInfo) {
  const diagnostics = environmentDiagnostics(row)
  const problem = diagnostics.find((item) => item.status !== 'HEALTHY')
  if (problem) return { message: problem.message, nextStep: problem.nextStep }
  if (row.status === 'HEALTHY') return { message: '当前未发现问题', nextStep: '可用于后续服务关联、发布和部署' }
  return {
    message: '环境待验证，尚未发现明确缺失项',
    nextStep: '请在详情中执行连接测试；如仍待验证，请到基础资源刷新探测结果',
  }
}

function environmentDiagnostics(row: EnvironmentInfo): EnvironmentDiagnostic[] {
  const latestChecks = checkResultsByEnvironmentId.value[row.id]
  if (latestChecks) return latestChecks
  const diagnostics: EnvironmentDiagnostic[] = []
  if (row.type === 'PROJECT') {
    diagnostics.push({
      component: 'Agent',
      status: row.agentStatus === 'ONLINE' ? 'HEALTHY' : 'DEGRADED',
      message: row.agentStatus === 'ONLINE' ? '远程 Agent 在线' : `远程 Agent 当前状态为 ${row.agentStatus || '未知'}`,
      nextStep: row.agentStatus === 'ONLINE'
        ? '可继续校验 Harbor/Jenkins 资源范围'
        : '请启动并注册该环境 Agent，确认 Agent 配置的环境 ID 与详情中的 Agent 环境 ID 一致',
    })
  }
  if (row.type === 'LOCAL') {
    appendScopeDiagnostics(diagnostics, row, 'K8S')
  }
  appendScopeDiagnostics(diagnostics, row, 'HARBOR')
  appendScopeDiagnostics(diagnostics, row, 'JENKINS')
  return diagnostics
}

function appendScopeDiagnostics(
  output: EnvironmentDiagnostic[],
  row: EnvironmentInfo,
  resourceType: EnvironmentResourceBinding['resourceType'],
) {
  for (const binding of bindingsForDiagnostics(row, resourceType)) {
    const availableValues = availableScopes(resourceType, binding.resourceId)
    const label = scopeLabel(resourceType)
    const exists = availableValues.includes(binding.scopeValue)
    output.push({
      component: label,
      status: exists ? 'HEALTHY' : 'DEGRADED',
      message: exists
        ? `${label} ${binding.scopeValue} 已在最近探测结果中发现`
        : `${label} ${binding.scopeValue} 未在最近探测结果中发现`,
      nextStep: exists ? '无需处理' : missingScopeNextStep(resourceType),
    })
  }
}

function bindingsForDiagnostics(row: EnvironmentInfo, resourceType: EnvironmentResourceBinding['resourceType']) {
  const bindings = row.bindings?.filter((item) => item.resourceType === resourceType) ?? []
  if (bindings.length > 0) return bindings
  if (resourceType === 'K8S' && row.clusterId && row.namespace) {
    return [{ resourceType, resourceId: row.clusterId, scopeType: 'NAMESPACE' as const, scopeValue: row.namespace, isDefault: true }]
  }
  if (resourceType === 'HARBOR' && row.registryId && row.registryProject) {
    return [{ resourceType, resourceId: row.registryId, scopeType: 'PROJECT' as const, scopeValue: row.registryProject, isDefault: true }]
  }
  if (resourceType === 'JENKINS' && row.jenkinsId && row.jenkinsView) {
    return [{ resourceType, resourceId: row.jenkinsId, scopeType: 'VIEW' as const, scopeValue: row.jenkinsView, isDefault: true }]
  }
  return []
}

function availableScopes(resourceType: EnvironmentResourceBinding['resourceType'], resourceId: string) {
  if (resourceType === 'K8S') return kubernetesClusters.value.find((item) => item.id === resourceId)?.namespaces ?? []
  if (resourceType === 'HARBOR') return harborRegistries.value.find((item) => item.id === resourceId)?.projects ?? []
  return jenkinsInstances.value.find((item) => item.id === resourceId)?.views ?? []
}

function scopeLabel(resourceType: EnvironmentResourceBinding['resourceType']) {
  if (resourceType === 'K8S') return 'K8s 命名空间'
  if (resourceType === 'HARBOR') return 'Harbor 镜像项目'
  return 'Jenkins 流水线视图'
}

function missingScopeNextStep(resourceType: EnvironmentResourceBinding['resourceType']) {
  if (resourceType === 'K8S') return '请到基础资源刷新 K8s 探测；若仍不存在，请在集群中创建命名空间或修改环境绑定'
  if (resourceType === 'HARBOR') return '请到基础资源刷新 Harbor 探测；若仍不存在，请在 Harbor 创建镜像项目或修改环境绑定'
  return '请到基础资源刷新 Jenkins 探测；若仍不存在，请在 Jenkins 创建流水线视图或修改环境绑定'
}

function applyEnvironmentTypeDefaults(type: EnvironmentPayload['type']) {
  if (type === 'PROJECT') {
    form.value.networkMode = 'AGENT'
    form.value.clusterId = ''
    form.value.namespace = ''
    form.value.namespaces = []
    form.value.registryId ||= harborRegistries.value[0]?.id || ''
    form.value.jenkinsId ||= jenkinsInstances.value[0]?.id || ''
    return
  }
  form.value.networkMode = 'DIRECT'
  form.value.clusterId ||= kubernetesClusters.value[0]?.id || ''
  form.value.registryId ||= harborRegistries.value[0]?.id || ''
  form.value.jenkinsId ||= jenkinsInstances.value[0]?.id || ''
}

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [environmentItems, clusterItems, registryItems, jenkinsItems] = await Promise.all([
      listEnvironments(),
      listKubernetesClusters(),
      listHarborRegistries(),
      listJenkinsInstances(),
    ])
    environments.value = environmentItems
    kubernetesClusters.value = clusterItems
    harborRegistries.value = registryItems
    jenkinsInstances.value = jenkinsItems
    if (dialogVisible.value && dialogMode.value === 'create') {
      applyEnvironmentTypeDefaults(form.value.type)
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '环境管理数据加载失败'
  } finally {
    loading.value = false
  }
}

function openDrawer(row: EnvironmentInfo) {
  activeEnvironment.value = row
  drawerVisible.value = true
}

function openCreateDialog() {
  dialogMode.value = 'create'
  form.value = emptyEnvironmentForm()
  applyEnvironmentTypeDefaults(form.value.type)
  dialogVisible.value = true
}

function openEditDialog(row: EnvironmentInfo) {
  dialogMode.value = 'edit'
  form.value = environmentToForm(row)
  dialogVisible.value = true
}

async function submitEnvironment() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写环境名称')
    return
  }
  if (form.value.type === 'LOCAL' && (!form.value.clusterId || normalizeScopes(form.value.namespaces, form.value.namespace).length === 0)) {
    ElMessage.warning('请完整选择 K8s 集群、Harbor 仓库，并填写命名空间与镜像项目')
    return
  }
  if (!form.value.registryId || normalizeScopes(form.value.registryProjects, form.value.registryProject).length === 0) {
    ElMessage.warning('请完整选择 Harbor 仓库并填写镜像项目')
    return
  }

  submitting.value = true
  try {
    const payload = trimEnvironmentPayload(form.value)
    const missingScopes = missingScopesBeforeSave()
    if (missingScopes.length > 0) {
      ElMessage.warning(`存在未在最近探测结果中发现的资源范围：${missingScopes.join('、')}，环境将保存为需验证状态`)
    }
    let savedEnvironment: EnvironmentInfo
    if (dialogMode.value === 'create') {
      savedEnvironment = await createEnvironment(payload)
    } else {
      savedEnvironment = await updateEnvironment(form.value.id, { ...payload, status: form.value.status })
    }
    if (savedEnvironment.status === 'DEGRADED' || missingScopes.length > 0) {
      ElMessage.warning('环境已保存，但存在未验证的资源范围，请刷新探测或执行连接测试后再用于发布/部署')
    } else {
      ElMessage.success(dialogMode.value === 'create' ? '环境已创建' : '环境已更新')
    }
    dialogVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '环境保存失败')
  } finally {
    submitting.value = false
  }
}

function environmentToForm(row: EnvironmentInfo): EnvironmentForm {
  return {
    ...emptyEnvironmentForm(),
    ...row,
    namespaces: scopedValuesFromBindings(row, 'K8S', row.namespace),
    registryProjects: scopedValuesFromBindings(row, 'HARBOR', row.registryProject),
    jenkinsViews: scopedValuesFromBindings(row, 'JENKINS', row.jenkinsView),
  }
}

function scopedValuesFromBindings(row: EnvironmentInfo, resourceType: EnvironmentResourceBinding['resourceType'], fallback: string) {
  const bindings = row.bindings?.filter((item) => item.resourceType === resourceType) ?? []
  if (bindings.length === 0) return fallback ? [fallback] : []
  const defaultBinding = bindings.find((item) => item.isDefault)
  const ordered = defaultBinding ? [defaultBinding, ...bindings.filter((item) => item !== defaultBinding)] : bindings
  return normalizeScopes(ordered.map((item) => item.scopeValue))
}

function trimEnvironmentPayload(payload: EnvironmentForm): EnvironmentPayload {
  const type = payload.type === 'LOCAL' ? 'LOCAL' : 'PROJECT'
  const networkMode = type === 'LOCAL' ? 'DIRECT' : 'AGENT'
  const namespaces = type === 'LOCAL' ? normalizeScopes(payload.namespaces, payload.namespace) : []
  const registryProjects = normalizeScopes(payload.registryProjects, payload.registryProject)
  const jenkinsViews = normalizeScopes(payload.jenkinsViews, payload.jenkinsView)
  const resourcePayload = {
    clusterId: type === 'LOCAL' ? payload.clusterId.trim() : '',
    namespace: namespaces[0] ?? '',
    registryId: payload.registryId.trim(),
    registryProject: registryProjects[0] ?? '',
    jenkinsId: payload.jenkinsId.trim(),
    jenkinsView: jenkinsViews[0] ?? '',
  }
  return {
    id: payload.id.trim(),
    name: payload.name.trim(),
    code: payload.code.trim() || generateEnvironmentCode(payload.name, type),
    type,
    deployTargetType: payload.deployTargetType === 'DOCKER_COMPOSE' ? 'DOCKER_COMPOSE' : 'KUBERNETES',
    networkMode,
    ...resourcePayload,
    bindings: buildBindings(type, resourcePayload, {
      namespaces,
      registryProjects,
      jenkinsViews,
    }),
  }
}

function buildBindings(
  type: EnvironmentPayload['type'],
  payload: Pick<EnvironmentPayload, 'clusterId' | 'registryId' | 'jenkinsId'>,
  scopes: Pick<EnvironmentForm, 'namespaces' | 'registryProjects' | 'jenkinsViews'>,
) {
  const bindings: EnvironmentPayload['bindings'] = []
  if (type === 'LOCAL' && payload.clusterId) {
    scopes.namespaces.forEach((namespace, index) => bindings.push({
      resourceType: 'K8S',
      resourceId: payload.clusterId,
      scopeType: 'NAMESPACE',
      scopeValue: namespace,
      isDefault: index === 0,
    }))
  }
  if (payload.registryId) {
    scopes.registryProjects.forEach((project, index) => bindings.push({
      resourceType: 'HARBOR',
      resourceId: payload.registryId,
      scopeType: 'PROJECT',
      scopeValue: project,
      isDefault: index === 0,
    }))
  }
  if (payload.jenkinsId) {
    scopes.jenkinsViews.forEach((view, index) => bindings.push({
      resourceType: 'JENKINS',
      resourceId: payload.jenkinsId,
      scopeType: 'VIEW',
      scopeValue: view,
      isDefault: index === 0,
    }))
  }
  return bindings
}

function normalizeScopes(values: string[], fallback = '') {
  const uniqueValues: string[] = []
  for (const value of [...values, fallback]) {
    const scope = value.trim()
    if (scope && !uniqueValues.includes(scope)) uniqueValues.push(scope)
  }
  return uniqueValues
}

function missingScopesBeforeSave() {
  const missingScopes: string[] = []
  if (form.value.type === 'LOCAL') {
    appendMissingScopes(missingScopes, 'K8s 命名空间', normalizeScopes(form.value.namespaces, form.value.namespace), selectedNamespaces.value)
  }
  appendMissingScopes(missingScopes, 'Harbor 镜像项目', normalizeScopes(form.value.registryProjects, form.value.registryProject), selectedProjects.value)
  if (form.value.jenkinsId) {
    appendMissingScopes(missingScopes, 'Jenkins 流水线视图', normalizeScopes(form.value.jenkinsViews, form.value.jenkinsView), selectedViews.value)
  }
  return missingScopes
}

function appendMissingScopes(output: string[], label: string, values: string[], availableValues: string[]) {
  const availableSet = new Set(availableValues.map((item) => item.trim()).filter(Boolean))
  for (const value of values) {
    if (!availableSet.has(value)) {
      output.push(`${label} ${value}`)
    }
  }
}

async function handleCheckEnvironment(id: string) {
  checkingEnvironment.value = true
  try {
    const result = await checkEnvironment(id)
    checkResultsByEnvironmentId.value[id] = diagnosticsFromCheckResult(result)
    const problemCount = checkResultsByEnvironmentId.value[id].filter((item) => item.status !== 'HEALTHY').length
    if (problemCount > 0) {
      ElMessage.warning(`连接测试完成：发现 ${problemCount} 个问题，请查看详情诊断结果`)
    } else {
      ElMessage.success('连接测试完成：未发现问题')
    }
    await loadAll()
    activeEnvironment.value = environments.value.find((item) => item.id === id) ?? activeEnvironment.value
  } catch (error) {
    const message = error instanceof Error ? error.message : '连接测试失败'
    checkResultsByEnvironmentId.value[id] = [{
      component: '连接测试',
      status: 'DEGRADED',
      message: `连接测试失败：${message}`,
      nextStep: '请检查基础资源配置、远程 Agent 状态或后端集成配置后重新测试',
    }]
    ElMessage.error(message)
  } finally {
    checkingEnvironment.value = false
  }
}

function diagnosticsFromCheckResult(result: EnvironmentCheckResult): EnvironmentDiagnostic[] {
  return result.checks.map((item) => ({
    component: item.component || item.name || '连接测试',
    status: item.status === 'HEALTHY' ? 'HEALTHY' : item.status === 'DEGRADED' ? 'DEGRADED' : 'UNKNOWN',
    message: item.message || `${item.component || item.name || '连接测试'} 状态为 ${item.status}`,
    nextStep: item.status === 'HEALTHY' ? '无需处理' : nextStepForCheckMessage(`${item.component || item.name || ''} ${item.message}`),
  }))
}

function nextStepForCheckMessage(message = '') {
  if (message.includes('K8s') || message.includes('命名空间')) return missingScopeNextStep('K8S')
  if (message.includes('Harbor') || message.includes('镜像项目')) return missingScopeNextStep('HARBOR')
  if (message.includes('Jenkins') || message.includes('流水线视图')) return missingScopeNextStep('JENKINS')
  if (message.includes('Agent')) return '请启动并注册该环境 Agent，确认 Agent 配置的环境 ID 与详情中的 Agent 环境 ID 一致'
  return '请根据失败信息检查基础资源配置，修正后重新执行连接测试'
}

function generateEnvironmentCode(name: string, type: EnvironmentPayload['type']) {
  if (/[^\x00-\x7F]/.test(name)) return `${type === 'LOCAL' ? 'local' : 'remote'}-${timestampCode()}`
  const normalized = normalizeEnvironmentCode(name)
  if (normalized) return normalized
  return `${type === 'LOCAL' ? 'local' : 'remote'}-${timestampCode()}`
}

function normalizeEnvironmentCode(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function timestampCode() {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return [
    now.getFullYear(),
    pad(now.getMonth() + 1),
    pad(now.getDate()),
    pad(now.getHours()),
    pad(now.getMinutes()),
    pad(now.getSeconds()),
  ].join('')
}

watch(
  () => form.value.type,
  (type) => {
    applyEnvironmentTypeDefaults(type)
    if (dialogMode.value === 'create' && form.value.name.trim()) {
      form.value.code = generateEnvironmentCode(form.value.name, type)
    }
  },
)

watch(
  () => form.value.name,
  (name) => {
    if (dialogMode.value === 'create') {
      form.value.code = name.trim() ? generateEnvironmentCode(name, form.value.type) : ''
    }
  },
)

watch(
  () => form.value.jenkinsId,
  (jenkinsId) => {
    if (!jenkinsId) {
      form.value.jenkinsView = ''
      form.value.jenkinsViews = []
    }
  },
)

onMounted(loadAll)
</script>

<style scoped>
.head-actions,
.readiness-grid {
  display: flex;
  gap: 10px;
}

.readiness-grid {
  flex-direction: column;
}

.environment-alert {
  margin-bottom: 12px;
}

.diagnosis-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  line-height: 18px;
}

.diagnosis-cell strong {
  color: #2f3847;
  font-size: 13px;
  overflow-wrap: anywhere;
}

.diagnosis-cell span {
  color: #7a8294;
  font-size: 12px;
  overflow-wrap: anywhere;
}

.form-tip {
  color: #7a8294;
  font-size: 12px;
  line-height: 20px;
  margin-top: 4px;
}
</style>
