<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>产品管理</h1>
        <p>产品对应一个可部署范围，可绑定到项目，并关联后续服务、发布和部署。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" title="重新加载产品与基础资源探测结果，不执行连接测试" @click="loadAll">刷新</el-button>
        <el-button type="primary" @click="openCreateDialog">新增产品</el-button>
      </div>
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 主线：本地产品绑定本地 K8s、Harbor、Jenkins；远程产品绑定本地 Harbor、Jenkins，并在产品内配置远程 K8s 命名空间和远程 Harbor 项目。"
      />
      <el-alert
        type="info"
        :closable="false"
        title="这些资源范围用于后续服务关联：服务会在产品内选择构建来源、镜像来源和部署目标；Agent 配置只负责连接远程环境，不负责指定产品资源映射。"
      />
      <el-alert
        v-if="blockedProjectEnvironmentCount > 0"
        type="warning"
        :closable="false"
        :title="`${blockedProjectEnvironmentCount} 个远程产品 Agent 未就绪，远程发布/部署提交前会被阻断。`"
      />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="keyword" placeholder="搜索产品、项目、标识" clearable />
          <el-select v-model="environmentType" placeholder="全部产品类型" clearable>
            <el-option label="本地产品" value="LOCAL" />
            <el-option label="远程产品" value="PROJECT" />
          </el-select>
        </div>
      </div>
      <el-alert v-if="errorMessage" class="environment-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="environment-table">
        <el-table-column label="产品与状态" min-width="320">
          <template #default="{ row }">
            <div class="environment-cell">
              <div class="environment-title">
                <strong>{{ row.name }}</strong>
                <StatusTag :status="row.status" />
              </div>
              <div class="environment-meta">
                <span>{{ row.code }}</span>
                <span>{{ row.projectName || '未绑定项目' }}</span>
                <el-tag size="small" :type="productStatusTagType(row.productStatus)" effect="light">
                  {{ productStatusLabel(row.productStatus) }}
                </el-tag>
                <span>{{ environmentTypeLabel(row.type) }}</span>
                <span>{{ deployTargetTypeLabel(row.deployTargetType) }}</span>
              </div>
              <div class="environment-problem" :class="{ healthy: problemDiagnostics(row).length === 0 }">
                <strong>{{ problemSummary(row).title }}</strong>
              </div>
              <el-button
                v-if="refreshableProblemTargets(row).length > 0"
                link
                type="primary"
                class="inline-action"
                :loading="refreshingEnvironmentId === row.id"
                @click="handleRefreshProblemResources(row)"
              >
                刷新相关探测
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="资源范围" min-width="300">
          <template #default="{ row }">
            <div class="resource-cell">
              <div v-if="row.type === 'LOCAL'">
                <span>K8s</span>
                <strong>{{ scopedResourceText(row, 'k8s') }}</strong>
              </div>
              <div v-else>
                <span>远程 K8s</span>
                <strong>{{ scopedResourceText(row, 'k8s', 'RUNTIME_TARGET') }}</strong>
              </div>
              <div>
                <span>{{ row.type === 'LOCAL' ? 'Harbor' : '本地 Harbor' }}</span>
                <strong>{{ scopedResourceText(row, 'harbor') }}</strong>
              </div>
              <div>
                <span>{{ row.type === 'LOCAL' ? 'Jenkins' : '本地 Jenkins' }}</span>
                <strong>{{ scopedResourceText(row, 'jenkins') }}</strong>
              </div>
              <div v-if="row.type === 'PROJECT'">
                <span>远程 Harbor</span>
                <strong>{{ scopedResourceText(row, 'harbor', 'RUNTIME_TARGET') }}</strong>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Agent" width="110">
          <template #default="{ row }">
            <span v-if="row.type === 'LOCAL'">无需 Agent</span>
            <div v-else class="agent-cell">
              <StatusTag :status="row.agentStatus" />
              <span v-if="row.agentStatus !== 'ONLINE'">影响远程执行</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="120">
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

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增产品' : '编辑产品'" width="580px" destroy-on-close>
      <el-form :model="form" label-width="120px">
        <el-form-item label="产品名称" required><el-input v-model="form.name" placeholder="数据中台生产" /></el-form-item>
        <el-form-item label="所属项目">
          <el-select v-model="form.projectId" placeholder="可选择一个项目，也可暂不绑定" clearable filterable>
            <el-option v-for="item in activeProjects" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
          <div class="form-tip">未绑定项目的产品仍可维护资源范围，后续可在这里绑定到项目。</div>
        </el-form-item>
        <el-form-item v-if="dialogMode === 'edit'" label="产品标识">
          <el-input v-model="form.code" placeholder="保存时自动生成" disabled />
          <div class="form-tip">由系统自动生成，用于远程 Agent 配置；保存后系统生成产品 ID：env-产品标识</div>
        </el-form-item>
        <el-form-item label="产品类型" required><el-segmented v-model="form.type" :options="typeOptions" /></el-form-item>
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
            <div class="form-tip">可绑定多个命名空间，首个命名空间作为默认部署目标；手工输入未探测到的值后，产品会进入需验证状态。</div>
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
            <div class="form-tip">可绑定多个镜像项目，首个镜像项目作为默认值；手工输入未探测到的值后，产品会进入需验证状态。</div>
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
            <div class="form-tip">这里选择本地镜像来源；远程 Harbor 项目在下面单独配置，由 Agent 负责验证。</div>
          </el-form-item>
          <el-form-item label="远程命名空间">
            <div class="runtime-status-line">
              <StatusTag :status="runtimeComponentStatus('K8S')" />
              <span>{{ runtimeComponentMessage('K8S') }}</span>
            </div>
            <el-select
              v-model="form.runtimeNamespaces"
              placeholder="选择 Agent 上报的远程 K8s 命名空间"
              multiple
              filterable
              :disabled="!boundAgent(form.id) || runtimeNamespaceOptions.length === 0"
            >
              <el-option v-for="item in runtimeNamespaceOptions" :key="item" :label="item" :value="item" />
            </el-select>
            <div class="form-tip">这里维护产品与远程 K8s 命名空间的映射；候选项来自已绑定 Agent 的心跳上报。</div>
          </el-form-item>
          <el-form-item label="远程镜像项目">
            <div class="runtime-status-line">
              <StatusTag :status="runtimeComponentStatus('HARBOR')" />
              <span>{{ runtimeComponentMessage('HARBOR') }}</span>
            </div>
            <el-select
              v-model="form.runtimeRegistryProjects"
              placeholder="选择 Agent 上报的远程 Harbor 项目"
              multiple
              filterable
              :disabled="!boundAgent(form.id) || runtimeRegistryProjectOptions.length === 0"
            >
              <el-option v-for="item in runtimeRegistryProjectOptions" :key="item" :label="item" :value="item" />
            </el-select>
            <div class="form-tip">这里维护产品与远程 Harbor 项目的映射；候选项来自已绑定 Agent 的心跳上报。</div>
          </el-form-item>
        </template>
        <el-form-item label="Jenkins" required>
          <el-select v-model="form.jenkinsId" placeholder="选择本地 Jenkins 资源" filterable>
            <el-option v-for="item in jenkinsInstances" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.jenkinsId" label="流水线视图" required>
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
          <div class="form-tip">本地和远程产品都使用本地 Jenkins 做构建；可绑定多个视图，手工输入未探测到的值后，产品会进入需验证状态。</div>
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
  probeEnvironment,
  updateEnvironment,
  type EnvironmentCheckResult,
  type EnvironmentInfo,
  type EnvironmentPayload,
  type EnvironmentResourceBinding,
} from '@/api/environments'
import { listProjects, type ProjectInfo } from '@/api/projects'
import {
  listHarborRegistries,
  listJenkinsInstances,
  listKubernetesClusters,
  refreshHarborRegistry,
  refreshJenkinsInstance,
  refreshKubernetesCluster,
  type HarborRegistry,
  type IntegrationResource,
  type JenkinsInstance,
  type KubernetesCluster,
} from '@/api/integrationResources'
import { listAgents, type AgentInfo, type AgentRuntimeComponentStatus } from '@/api/agents'

const keyword = ref('')
const environmentType = ref('')
const drawerVisible = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const activeEnvironment = ref<EnvironmentInfo | null>(null)
const environments = ref<EnvironmentInfo[]>([])
const agents = ref<AgentInfo[]>([])
const projects = ref<ProjectInfo[]>([])
const kubernetesClusters = ref<KubernetesCluster[]>([])
const harborRegistries = ref<HarborRegistry[]>([])
const jenkinsInstances = ref<JenkinsInstance[]>([])
const loading = ref(false)
const submitting = ref(false)
const checkingEnvironment = ref(false)
const refreshingEnvironmentId = ref('')
const errorMessage = ref('')
type EnvironmentDiagnostic = {
  component: string
  status: 'HEALTHY' | 'DEGRADED' | 'UNKNOWN'
  message: string
  nextStep: string
  resourceType?: EnvironmentResourceBinding['resourceType']
  resourceId?: string
}
type EnvironmentForm = EnvironmentPayload & {
  namespaces: string[]
  registryProjects: string[]
  jenkinsViews: string[]
  runtimeNamespaces: string[]
  runtimeRegistryProjects: string[]
}
const checkResultsByEnvironmentId = ref<Record<string, EnvironmentDiagnostic[]>>({})
const runtimeK8sResourceId = 'agent-runtime-k8s'
const runtimeHarborResourceId = 'agent-runtime-harbor'

const form = ref<EnvironmentForm>(emptyEnvironmentForm())

const typeOptions = [
  { label: '远程产品', value: 'PROJECT' },
  { label: '本地产品', value: 'LOCAL' },
]

const blockedProjectEnvironmentCount = computed(
  () => environments.value.filter((item) => item.type === 'PROJECT' && item.agentStatus !== 'ONLINE').length,
)

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return environments.value.filter((item) => {
    const keywordMatched = !q || `${item.name} ${item.code} ${item.projectName}`.toLowerCase().includes(q)
    const typeMatched = !environmentType.value || item.type === environmentType.value
    return keywordMatched && typeMatched
  })
})

const activeProjects = computed(() => projects.value.filter((item) => item.status !== 'DISABLED'))
const selectedNamespaces = computed(() => kubernetesClusters.value.find((item) => item.id === form.value.clusterId)?.namespaces ?? [])
const selectedProjects = computed(() => harborRegistries.value.find((item) => item.id === form.value.registryId)?.projects ?? [])
const selectedViews = computed(() => jenkinsInstances.value.find((item) => item.id === form.value.jenkinsId)?.views ?? [])
const runtimeNamespaceOptions = computed(() => runtimeScopeOptions('K8S', form.value.id))
const runtimeRegistryProjectOptions = computed(() => runtimeScopeOptions('HARBOR', form.value.id))
const activeDiagnostics = computed(() => {
  if (!activeEnvironment.value) return []
  return environmentDiagnostics(activeEnvironment.value)
})
const activeCheckHelpText = computed(() => {
  if (!activeEnvironment.value) return ''
  if (activeEnvironment.value.type === 'LOCAL') {
    return '本地连接测试：平台后端直接校验已绑定的 K8s、Harbor、Jenkins 范围，不依赖 Agent。'
  }
  return '远程连接测试：校验 Agent 在线状态，并由 Agent 探测项目环境 K8s 命名空间和 Harbor 镜像项目；Jenkins 由平台后端直连本地资源。'
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
    projectId: '',
    productStatus: 'UNBOUND',
    bindings: [],
    namespaces: [],
    registryProjects: [],
    jenkinsViews: [],
    runtimeNamespaces: [],
    runtimeRegistryProjects: [],
  }
}

function resourceName(items: IntegrationResource[], id: string) {
  if (!id) return '-'
  return items.find((item) => item.id === id)?.name || id
}

function environmentTypeLabel(type: string) {
  return type === 'LOCAL' ? '本地产品' : '远程产品'
}

function deployTargetTypeLabel(type: string) {
  return type === 'DOCKER_COMPOSE' ? 'docker-compose' : 'Kubernetes'
}

function productStatusLabel(status: string) {
  if (status === 'BOUND') return '已绑定项目'
  if (status === 'DISABLED') return '项目不可用'
  return '未绑定项目'
}

function productStatusTagType(status: string): '' | 'success' | 'info' | 'warning' | 'danger' | 'primary' {
  if (status === 'BOUND') return 'success'
  if (status === 'DISABLED') return 'info'
  return 'warning'
}

function scopedResourceText(row: EnvironmentInfo, resourceType: 'k8s' | 'harbor' | 'jenkins', bindingRole: EnvironmentResourceBinding['bindingRole'] = 'BUILD_SOURCE') {
  if (row.type === 'PROJECT' && resourceType === 'k8s' && bindingRole === 'BUILD_SOURCE') return 'Agent 执行'
  const type = bindingResourceType(resourceType)
  const bindings = row.bindings?.filter((item) => item.resourceType === type && bindingRoleOf(item) === bindingRole) ?? []
  if (bindings.length > 0) {
    const defaultBinding = bindings.find((item) => item.isDefault) ?? bindings[0]
    const orderedBindings = [defaultBinding, ...bindings.filter((item) => item !== defaultBinding)]
    const resourceIds = Array.from(new Set(orderedBindings.map((item) => item.resourceId)))
    if (resourceIds.length === 1) {
      return `${resourceDisplayName(type, defaultBinding.resourceId)} / ${orderedBindings.map((item) => item.scopeValue).join('、')}`
    }
    return orderedBindings.map((item) => `${resourceDisplayName(type, item.resourceId)} / ${item.scopeValue}`).join('；')
  }
  if (bindingRole === 'RUNTIME_TARGET') return '未配置'
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
  if (resourceType === 'K8S' && resourceId === runtimeK8sResourceId) return 'Agent 远程 K8s'
  if (resourceType === 'HARBOR' && resourceId === runtimeHarborResourceId) return 'Agent 远程 Harbor'
  if (resourceType === 'K8S') return resourceName(kubernetesClusters.value, resourceId)
  if (resourceType === 'HARBOR') return resourceName(harborRegistries.value, resourceId)
  return resourceName(jenkinsInstances.value, resourceId)
}

function drawerResourceName(resourceType: EnvironmentResourceBinding['resourceType'], resourceId: string) {
  return resourceDisplayName(resourceType, resourceId)
}

function bindingRoleOf(binding: EnvironmentResourceBinding) {
  return binding.bindingRole === 'RUNTIME_TARGET' ? 'RUNTIME_TARGET' : 'BUILD_SOURCE'
}

function problemDiagnostics(row: EnvironmentInfo) {
  return environmentDiagnostics(row).filter((item) => item.status !== 'HEALTHY')
}

function problemSummary(row: EnvironmentInfo) {
  const problems = problemDiagnostics(row)
  if (problems.length > 0) {
    return {
      title: problems.map((item) => item.message).join('；'),
      nextStep: compactNextStep(problems),
    }
  }
  return {
    title: row.status === 'HEALTHY' ? '当前未发现问题' : '产品待验证，尚未发现明确缺失项',
    nextStep: row.status === 'HEALTHY' ? '可用于后续服务关联、发布和部署' : '请在详情中执行连接测试；顶部刷新只重新加载产品与基础资源探测结果',
  }
}

function compactNextStep(problems: EnvironmentDiagnostic[]) {
  const steps = Array.from(new Set(problems.map((item) => item.nextStep).filter(Boolean)))
  if (steps.length === 0) return '请检查产品绑定和基础资源探测结果'
  if (steps.length === 1) return steps[0]
  const hasAgentProblem = problems.some((item) => item.component === 'Agent')
  const hasResourceProblem = problems.some((item) => item.resourceType && item.resourceId)
  if (hasAgentProblem && hasResourceProblem) return '请先启动或注册 Agent，并刷新相关基础资源探测'
  if (hasResourceProblem) return '请刷新相关基础资源探测；若仍不存在，请创建资源范围或修改产品绑定'
  return steps.join('；')
}

function environmentDiagnostics(row: EnvironmentInfo): EnvironmentDiagnostic[] {
  const latestChecks = checkResultsByEnvironmentId.value[row.id]
  if (latestChecks) return latestChecks
  return [...agentDiagnostics(row), ...resourceDiagnostics(row)]
}

function agentDiagnostics(row: EnvironmentInfo): EnvironmentDiagnostic[] {
  const diagnostics: EnvironmentDiagnostic[] = []
  if (row.type === 'PROJECT') {
    diagnostics.push({
      component: 'Agent',
      status: row.agentStatus === 'ONLINE' ? 'HEALTHY' : 'DEGRADED',
      message: row.agentStatus === 'ONLINE' ? '远程 Agent 在线' : `远程 Agent ${agentStatusText(row.agentStatus)}，会影响远程发布/部署执行`,
      nextStep: row.agentStatus === 'ONLINE'
        ? '可继续校验远程产品 K8s 命名空间和 Harbor 镜像项目'
        : '请启动并注册该产品 Agent，确认 Agent 配置的环境 ID 与详情中的 Agent 环境 ID 一致',
    })
  }
  return diagnostics
}

function resourceDiagnostics(row: EnvironmentInfo): EnvironmentDiagnostic[] {
  const diagnostics: EnvironmentDiagnostic[] = []
  if (row.type === 'LOCAL') appendScopeDiagnostics(diagnostics, row, 'K8S')
  appendScopeDiagnostics(diagnostics, row, 'HARBOR')
  appendScopeDiagnostics(diagnostics, row, 'JENKINS')
  if (row.type === 'PROJECT') appendRuntimeTargetDiagnostics(diagnostics, row)
  return diagnostics
}

function agentStatusText(status = '') {
  if (status === 'ONLINE') return '在线'
  if (status === 'OFFLINE') return '离线'
  if (status === 'UNBOUND') return '未绑定'
  if (status === 'NOT_REQUIRED') return '不需要'
  return status || '未知'
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
      resourceType,
      resourceId: binding.resourceId,
    })
  }
}

function appendRuntimeTargetDiagnostics(output: EnvironmentDiagnostic[], row: EnvironmentInfo) {
  const runtimeAgent = boundAgent(row.id)
  const runtimeK8sBindings = runtimeBindings(row, 'K8S')
  const runtimeHarborBindings = runtimeBindings(row, 'HARBOR')
  appendRuntimeComponentDiagnostic(output, row, 'K8S', runtimeAgent?.runtimeStatus?.kubernetes)
  appendRuntimeComponentDiagnostic(output, row, 'HARBOR', runtimeAgent?.runtimeStatus?.harbor)
  if (runtimeK8sBindings.length === 0) {
    output.push({
      component: '远程 K8s',
      status: 'DEGRADED',
      message: '未选择远程 K8s 命名空间',
      nextStep: '请编辑产品，从 Agent 上报的命名空间中选择该产品使用的范围；如果没有候选项，请检查 Agent 的 kubeconfig 配置',
    })
  }
  if (runtimeHarborBindings.length === 0) {
    output.push({
      component: '远程 Harbor',
      status: 'DEGRADED',
      message: '未选择远程 Harbor 项目',
      nextStep: '请编辑产品，从 Agent 上报的 Harbor 项目中选择该产品使用的范围；如果没有候选项，请检查 Agent 的 Harbor 配置',
    })
  }
}

function appendRuntimeComponentDiagnostic(
  output: EnvironmentDiagnostic[],
  row: EnvironmentInfo,
  resourceType: 'K8S' | 'HARBOR',
  component?: AgentRuntimeComponentStatus,
) {
  const label = resourceType === 'K8S' ? '远程 K8s' : '远程 Harbor'
  if (!boundAgent(row.id)) {
    output.push({
      component: label,
      status: 'DEGRADED',
      message: `${label} 未绑定 Agent，无法获取远程资源清单`,
      nextStep: '请先在 Agent 管理中把远程 Agent 绑定到该产品',
    })
    return
  }
  if (!component?.status) {
    output.push({
      component: label,
      status: 'UNKNOWN',
      message: `${label} 等待 Agent 上报状态`,
      nextStep: '请确认 Agent 已重启到最新版本，并点击刷新查看心跳上报结果',
    })
    return
  }
  output.push({
    component: label,
    status: runtimeDiagnosticStatus(component.status),
    message: `${label} ${runtimeComponentStatusText(component.status)}：${component.message || '无状态说明'}`,
    nextStep: component.status === 'HEALTHY' ? '无需处理' : runtimeComponentNextStep(resourceType),
  })
}

function refreshableProblemTargets(row: EnvironmentInfo) {
  const targetKeys = new Set<string>()
  const targets: Array<{ resourceType: EnvironmentResourceBinding['resourceType']; resourceId: string }> = []
  for (const item of problemDiagnostics(row)) {
    if (!item.resourceType || !item.resourceId) continue
    const key = `${item.resourceType}:${item.resourceId}`
    if (targetKeys.has(key)) continue
    targetKeys.add(key)
    targets.push({ resourceType: item.resourceType, resourceId: item.resourceId })
  }
  return targets
}

function bindingsForDiagnostics(row: EnvironmentInfo, resourceType: EnvironmentResourceBinding['resourceType']) {
  const bindings = row.bindings?.filter((item) => item.resourceType === resourceType && bindingRoleOf(item) === 'BUILD_SOURCE') ?? []
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

function runtimeBindings(row: EnvironmentInfo, resourceType: 'K8S' | 'HARBOR') {
  return row.bindings?.filter((item) => item.resourceType === resourceType && bindingRoleOf(item) === 'RUNTIME_TARGET') ?? []
}

function runtimeDiagnosticStatus(status = ''): EnvironmentDiagnostic['status'] {
  if (status === 'HEALTHY') return 'HEALTHY'
  if (status === 'UNHEALTHY') return 'DEGRADED'
  return 'UNKNOWN'
}

function runtimeComponentStatusText(status = '') {
  if (status === 'HEALTHY') return '正常'
  if (status === 'UNHEALTHY') return '异常'
  return '未知'
}

function runtimeComponentNextStep(resourceType: 'K8S' | 'HARBOR') {
  if (resourceType === 'K8S') return '请检查远程 Agent 配置中的 kubeconfig 是否可访问项目环境 K8s'
  return '请检查远程 Agent 配置中的 Harbor 地址、账号和网络连通性'
}

function runtimeScopeOptions(resourceType: 'K8S' | 'HARBOR', environmentId = form.value.id) {
  const component = runtimeComponent(resourceType, environmentId)
  const savedValues = resourceType === 'K8S' ? form.value.runtimeNamespaces : form.value.runtimeRegistryProjects
  return normalizeScopes([...(component?.items ?? []), ...savedValues])
}

function runtimeComponent(resourceType: 'K8S' | 'HARBOR', environmentId = form.value.id) {
  const agent = boundAgent(environmentId)
  return resourceType === 'K8S' ? agent?.runtimeStatus?.kubernetes : agent?.runtimeStatus?.harbor
}

function runtimeComponentStatus(resourceType: 'K8S' | 'HARBOR') {
  return runtimeComponent(resourceType)?.status || 'UNKNOWN'
}

function runtimeComponentMessage(resourceType: 'K8S' | 'HARBOR') {
  const label = resourceType === 'K8S' ? '远程 K8s' : '远程 Harbor'
  if (!form.value.id) return `保存产品并绑定 Agent 后，${label} 清单将由 Agent 上报`
  const agent = boundAgent(form.value.id)
  if (!agent) return `该产品尚未绑定 Agent，无法获取${label}清单`
  const component = runtimeComponent(resourceType)
  if (!component?.status) return `${label} 等待 Agent 上报`
  return component.message || `${label} ${runtimeComponentStatusText(component.status)}`
}

function boundAgent(environmentId: string) {
  if (!environmentId) return undefined
  return agents.value.find((item) => item.environmentId === environmentId)
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
  if (resourceType === 'K8S') return '请到基础资源刷新 K8s 探测；若仍不存在，请在集群中创建命名空间或修改产品绑定'
  if (resourceType === 'HARBOR') return '请到基础资源刷新 Harbor 探测；若仍不存在，请在 Harbor 创建镜像项目或修改产品绑定'
  return '请到基础资源刷新 Jenkins 探测；若仍不存在，请在 Jenkins 创建流水线视图或修改产品绑定'
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
    const [environmentItems, agentItems, projectItems, clusterItems, registryItems, jenkinsItems] = await Promise.all([
      listEnvironments(),
      listAgents(),
      listProjects(),
      listKubernetesClusters(),
      listHarborRegistries(),
      listJenkinsInstances(),
    ])
    environments.value = environmentItems
    agents.value = agentItems
    projects.value = projectItems
    kubernetesClusters.value = clusterItems
    harborRegistries.value = registryItems
    jenkinsInstances.value = jenkinsItems
    if (dialogVisible.value && dialogMode.value === 'create') {
      applyEnvironmentTypeDefaults(form.value.type)
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '产品管理数据加载失败'
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
    ElMessage.warning('请填写产品名称')
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
  if (!form.value.jenkinsId || normalizeScopes(form.value.jenkinsViews, form.value.jenkinsView).length === 0) {
    ElMessage.warning('请完整选择 Jenkins 并填写流水线视图')
    return
  }
  if (form.value.type === 'PROJECT') {
    if (normalizeScopes(form.value.runtimeNamespaces).length === 0) {
      ElMessage.warning('请选择 Agent 上报的远程 K8s 命名空间')
      return
    }
    if (normalizeScopes(form.value.runtimeRegistryProjects).length === 0) {
      ElMessage.warning('请选择 Agent 上报的远程 Harbor 项目')
      return
    }
  }

  submitting.value = true
  try {
    const payload = trimEnvironmentPayload(form.value)
    const missingScopes = missingScopesBeforeSave()
    if (missingScopes.length > 0) {
      ElMessage.warning(`存在未在最近探测结果中发现的资源范围：${missingScopes.join('、')}，产品将保存为需验证状态`)
    }
    let savedEnvironment: EnvironmentInfo
    if (dialogMode.value === 'create') {
      savedEnvironment = await createEnvironment(payload)
    } else {
      savedEnvironment = await updateEnvironment(form.value.id, { ...payload, status: form.value.status })
    }
    if (savedEnvironment.status === 'DEGRADED' || missingScopes.length > 0) {
      ElMessage.warning('产品已保存，但存在未验证的资源范围，请刷新探测或执行连接测试后再用于发布/部署')
    } else {
      ElMessage.success(dialogMode.value === 'create' ? '产品已创建' : '产品已更新')
    }
    dialogVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '产品保存失败')
  } finally {
    submitting.value = false
  }
}

function environmentToForm(row: EnvironmentInfo): EnvironmentForm {
  return {
    ...emptyEnvironmentForm(),
    ...row,
    namespaces: scopedValuesFromBindings(row, 'K8S', row.namespace, 'BUILD_SOURCE'),
    registryProjects: scopedValuesFromBindings(row, 'HARBOR', row.registryProject, 'BUILD_SOURCE'),
    jenkinsViews: scopedValuesFromBindings(row, 'JENKINS', row.jenkinsView, 'BUILD_SOURCE'),
    runtimeNamespaces: scopedValuesFromBindings(row, 'K8S', '', 'RUNTIME_TARGET'),
    runtimeRegistryProjects: scopedValuesFromBindings(row, 'HARBOR', '', 'RUNTIME_TARGET'),
  }
}

function scopedValuesFromBindings(
  row: EnvironmentInfo,
  resourceType: EnvironmentResourceBinding['resourceType'],
  fallback: string,
  bindingRole: EnvironmentResourceBinding['bindingRole'] = 'BUILD_SOURCE',
) {
  const bindings = row.bindings?.filter((item) => item.resourceType === resourceType && bindingRoleOf(item) === bindingRole) ?? []
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
  const runtimeNamespaces = type === 'PROJECT' ? normalizeScopes(payload.runtimeNamespaces) : []
  const runtimeRegistryProjects = type === 'PROJECT' ? normalizeScopes(payload.runtimeRegistryProjects) : []
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
    projectId: payload.projectId.trim(),
    productStatus: payload.projectId.trim() ? 'BOUND' : 'UNBOUND',
    ...resourcePayload,
    bindings: buildBindings(type, resourcePayload, {
      namespaces,
      registryProjects,
      jenkinsViews,
      runtimeNamespaces,
      runtimeRegistryProjects,
    }),
  }
}

function buildBindings(
  type: EnvironmentPayload['type'],
  payload: Pick<EnvironmentPayload, 'clusterId' | 'registryId' | 'jenkinsId'>,
  scopes: Pick<EnvironmentForm, 'namespaces' | 'registryProjects' | 'jenkinsViews' | 'runtimeNamespaces' | 'runtimeRegistryProjects'>,
) {
  const bindings: EnvironmentPayload['bindings'] = []
  if (type === 'LOCAL' && payload.clusterId) {
    scopes.namespaces.forEach((namespace, index) => bindings.push({
      resourceType: 'K8S',
      bindingRole: 'BUILD_SOURCE',
      resourceId: payload.clusterId,
      scopeType: 'NAMESPACE',
      scopeValue: namespace,
      isDefault: index === 0,
    }))
  }
  if (payload.registryId) {
    scopes.registryProjects.forEach((project, index) => bindings.push({
      resourceType: 'HARBOR',
      bindingRole: 'BUILD_SOURCE',
      resourceId: payload.registryId,
      scopeType: 'PROJECT',
      scopeValue: project,
      isDefault: index === 0,
    }))
  }
  if (payload.jenkinsId) {
    scopes.jenkinsViews.forEach((view, index) => bindings.push({
      resourceType: 'JENKINS',
      bindingRole: 'BUILD_SOURCE',
      resourceId: payload.jenkinsId,
      scopeType: 'VIEW',
      scopeValue: view,
      isDefault: index === 0,
    }))
  }
  if (type === 'PROJECT') {
    scopes.runtimeNamespaces.forEach((namespace, index) => bindings.push({
      resourceType: 'K8S',
      bindingRole: 'RUNTIME_TARGET',
      resourceId: runtimeK8sResourceId,
      scopeType: 'NAMESPACE',
      scopeValue: namespace,
      isDefault: index === 0,
    }))
    scopes.runtimeRegistryProjects.forEach((project, index) => bindings.push({
      resourceType: 'HARBOR',
      bindingRole: 'RUNTIME_TARGET',
      resourceId: runtimeHarborResourceId,
      scopeType: 'PROJECT',
      scopeValue: project,
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
  if (form.value.type === 'PROJECT') {
    appendMissingScopes(missingScopes, '远程 K8s 命名空间', normalizeScopes(form.value.runtimeNamespaces), runtimeNamespaceOptions.value)
    appendMissingScopes(missingScopes, '远程 Harbor 项目', normalizeScopes(form.value.runtimeRegistryProjects), runtimeRegistryProjectOptions.value)
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
    const currentEnvironment = environments.value.find((item) => item.id === id) ?? activeEnvironment.value
    if (currentEnvironment?.type === 'PROJECT') {
      const result = await probeEnvironment(id)
      checkResultsByEnvironmentId.value[id] = [{
        component: '远程运行资源',
        status: 'UNKNOWN',
        message: result.message || '远程探测任务已下发，等待 Agent 回传结果',
        nextStep: '请稍后点击刷新查看 Agent 回传后的产品状态；若长时间没有变化，请检查 Agent 是否在线',
      }]
      ElMessage.success('远程探测任务已下发，等待 Agent 回传结果')
      await loadAll()
      activeEnvironment.value = environments.value.find((item) => item.id === id) ?? activeEnvironment.value
      return
    }
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

async function handleRefreshProblemResources(row: EnvironmentInfo) {
  const targets = refreshableProblemTargets(row)
  if (targets.length === 0) {
    ElMessage.warning('当前问题不需要刷新基础资源探测')
    return
  }
  refreshingEnvironmentId.value = row.id
  try {
    await Promise.all(targets.map((target) => refreshProblemTarget(target.resourceType, target.resourceId)))
    ElMessage.success('相关基础资源探测已刷新')
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '刷新探测失败')
  } finally {
    refreshingEnvironmentId.value = ''
  }
}

async function refreshProblemTarget(resourceType: EnvironmentResourceBinding['resourceType'], resourceId: string) {
  if (resourceType === 'K8S') {
    await refreshKubernetesCluster(resourceId)
  } else if (resourceType === 'HARBOR') {
    await refreshHarborRegistry(resourceId)
  } else {
    await refreshJenkinsInstance(resourceId)
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
  if (message.includes('Agent')) return '请启动并注册该产品 Agent，确认 Agent 配置的环境 ID 与详情中的 Agent 环境 ID 一致'
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

.environment-table :deep(.cell) {
  overflow-wrap: anywhere;
}

.environment-cell,
.resource-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.environment-title {
  align-items: center;
  display: flex;
  gap: 8px;
  justify-content: space-between;
}

.environment-title strong {
  color: #2f3847;
  font-size: 14px;
  overflow-wrap: anywhere;
}

.environment-meta {
  color: #7a8294;
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  font-size: 12px;
  line-height: 18px;
}

.environment-problem {
  border-left: 3px solid #e6a23c;
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 18px;
  padding-left: 8px;
}

.environment-problem.healthy {
  border-left-color: #67c23a;
}

.environment-problem strong {
  color: #2f3847;
  font-size: 13px;
  overflow-wrap: anywhere;
}

.inline-action {
  align-self: flex-start;
  padding: 0;
}

.resource-cell div {
  display: grid;
  gap: 8px;
  grid-template-columns: 62px minmax(0, 1fr);
  line-height: 18px;
}

.resource-cell span {
  color: #7a8294;
  font-size: 12px;
}

.resource-cell strong {
  color: #2f3847;
  font-size: 12px;
  font-weight: 500;
  overflow-wrap: anywhere;
}

.agent-cell {
  align-items: flex-start;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.agent-cell span:last-child {
  color: #7a8294;
  font-size: 12px;
  line-height: 16px;
}

.form-tip {
  color: #7a8294;
  font-size: 12px;
  line-height: 20px;
  margin-top: 4px;
}

.runtime-placeholder {
  background: #f6f8fb;
  border: 1px dashed #d8dee8;
  border-radius: 6px;
  color: #606a7b;
  font-size: 13px;
  line-height: 20px;
  padding: 10px 12px;
  width: 100%;
}

.runtime-status-line {
  align-items: center;
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  min-height: 24px;
}

.runtime-status-line span:last-child {
  color: #606a7b;
  font-size: 12px;
  line-height: 18px;
}
</style>
