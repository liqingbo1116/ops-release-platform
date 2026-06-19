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
        title="V1 主线：本地环境关联 K8s namespace、Harbor project、Jenkins view；远程环境关联本地 Harbor project 与 Jenkins view，远程 K8s 由 Agent 执行。"
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
        <el-table-column label="K8s / namespace" min-width="210">
          <template #default="{ row }">{{ scopedResourceText(row, 'k8s') }}</template>
        </el-table-column>
        <el-table-column label="Harbor / project" min-width="210">
          <template #default="{ row }">{{ scopedResourceText(row, 'harbor') }}</template>
        </el-table-column>
        <el-table-column label="Jenkins / view" min-width="210">
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
      :checking="checkingEnvironment"
      @check="handleCheckEnvironment"
    />

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增环境' : '编辑环境'" width="580px" destroy-on-close>
      <el-form :model="form" label-width="120px">
        <el-form-item label="环境名称" required><el-input v-model="form.name" placeholder="项目 X 生产" /></el-form-item>
        <el-form-item label="环境标识">
          <el-input v-model="form.code" placeholder="保存时自动生成" disabled />
          <div class="form-tip">由系统自动生成，用于远程 Agent 配置；保存后系统生成环境 ID：env-环境标识</div>
        </el-form-item>
        <el-form-item label="环境类型" required><el-segmented v-model="form.type" :options="typeOptions" /></el-form-item>
        <el-form-item label="部署目标">
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
            <el-select v-model="form.namespace" placeholder="选择或输入 namespace" filterable allow-create default-first-option>
              <el-option v-for="item in selectedNamespaces" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
          <el-form-item label="Harbor 仓库" required>
            <el-select v-model="form.registryId" placeholder="先在基础资源维护 Harbor 仓库" filterable>
              <el-option v-for="item in harborRegistries" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="Harbor project" required>
            <el-select v-model="form.registryProject" placeholder="选择或输入 project" filterable allow-create default-first-option>
              <el-option v-for="item in selectedProjects" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="远程 K8s">
            <el-input value="由远程 Agent 连接并执行" disabled />
          </el-form-item>
          <el-form-item label="Harbor 仓库" required>
            <el-select v-model="form.registryId" placeholder="选择本地 Harbor 仓库" filterable>
              <el-option v-for="item in harborRegistries" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="Harbor project" required>
            <el-select v-model="form.registryProject" placeholder="选择或输入本地 project" filterable allow-create default-first-option>
              <el-option v-for="item in selectedProjects" :key="item" :label="item" :value="item" />
            </el-select>
          </el-form-item>
        </template>
        <el-form-item label="Jenkins">
          <el-select v-model="form.jenkinsId" placeholder="选择 Jenkins 资源" clearable filterable>
            <el-option v-for="item in jenkinsInstances" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Jenkins view">
          <el-select v-model="form.jenkinsView" placeholder="选择或输入 view" clearable filterable allow-create default-first-option>
            <el-option v-for="item in selectedViews" :key="item" :label="item" :value="item" />
          </el-select>
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
import { checkEnvironment, createEnvironment, listEnvironments, updateEnvironment, type EnvironmentInfo, type EnvironmentPayload } from '@/api/environments'
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
const form = ref<EnvironmentPayload>(emptyEnvironmentForm())

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

function emptyEnvironmentForm(): EnvironmentPayload {
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
  if (resourceType === 'k8s') return `${resourceName(kubernetesClusters.value, row.clusterId)} / ${row.namespace || '-'}`
  if (resourceType === 'harbor') return `${resourceName(harborRegistries.value, row.registryId)} / ${row.registryProject || '-'}`
  return `${resourceName(jenkinsInstances.value, row.jenkinsId)} / ${row.jenkinsView || '-'}`
}

function applyEnvironmentTypeDefaults(type: EnvironmentPayload['type']) {
  if (type === 'PROJECT') {
    form.value.networkMode = 'AGENT'
    form.value.clusterId = ''
    form.value.namespace = ''
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
  form.value = { ...row }
  dialogVisible.value = true
}

async function submitEnvironment() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写环境名称')
    return
  }
  if (form.value.type === 'LOCAL' && (!form.value.clusterId || !form.value.namespace.trim())) {
    ElMessage.warning('请完整选择 K8s/Harbor 资源并填写 namespace 与 project')
    return
  }
  if (!form.value.registryId || !form.value.registryProject.trim()) {
    ElMessage.warning('请完整选择 Harbor 仓库并填写 project')
    return
  }

  submitting.value = true
  try {
    const payload = trimEnvironmentPayload(form.value)
    if (dialogMode.value === 'create') {
      await createEnvironment(payload)
      ElMessage.success('环境已创建')
    } else {
      await updateEnvironment(form.value.id, { ...payload, status: form.value.status })
      ElMessage.success('环境已更新')
    }
    dialogVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '环境保存失败')
  } finally {
    submitting.value = false
  }
}

function trimEnvironmentPayload(payload: EnvironmentPayload): EnvironmentPayload {
  const type = payload.type === 'LOCAL' ? 'LOCAL' : 'PROJECT'
  const networkMode = type === 'LOCAL' ? 'DIRECT' : 'AGENT'
  const resourcePayload = {
    clusterId: type === 'LOCAL' ? payload.clusterId.trim() : '',
    namespace: type === 'LOCAL' ? payload.namespace.trim() : '',
    registryId: payload.registryId.trim(),
    registryProject: payload.registryProject.trim(),
    jenkinsId: payload.jenkinsId.trim(),
    jenkinsView: payload.jenkinsView.trim(),
  }
  return {
    ...payload,
    id: payload.id.trim(),
    name: payload.name.trim(),
    code: payload.code.trim() || generateEnvironmentCode(payload.name, type),
    type,
    deployTargetType: payload.deployTargetType === 'DOCKER_COMPOSE' ? 'DOCKER_COMPOSE' : 'KUBERNETES',
    networkMode,
    ...resourcePayload,
    bindings: buildDefaultBindings(type, resourcePayload),
  }
}

function buildDefaultBindings(type: EnvironmentPayload['type'], payload: Pick<EnvironmentPayload, 'clusterId' | 'namespace' | 'registryId' | 'registryProject' | 'jenkinsId' | 'jenkinsView'>) {
  const bindings: EnvironmentPayload['bindings'] = []
  if (type === 'LOCAL' && payload.clusterId && payload.namespace) {
    bindings.push({
      resourceType: 'K8S',
      resourceId: payload.clusterId,
      scopeType: 'NAMESPACE',
      scopeValue: payload.namespace,
      isDefault: true,
    })
  }
  if (payload.registryId && payload.registryProject) {
    bindings.push({
      resourceType: 'HARBOR',
      resourceId: payload.registryId,
      scopeType: 'PROJECT',
      scopeValue: payload.registryProject,
      isDefault: true,
    })
  }
  if (payload.jenkinsId && payload.jenkinsView) {
    bindings.push({
      resourceType: 'JENKINS',
      resourceId: payload.jenkinsId,
      scopeType: 'VIEW',
      scopeValue: payload.jenkinsView,
      isDefault: true,
    })
  }
  return bindings
}

async function handleCheckEnvironment(id: string) {
  checkingEnvironment.value = true
  try {
    const result = await checkEnvironment(id)
    ElMessage.success(`连接测试完成：${result.status}`)
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '连接测试失败')
  } finally {
    checkingEnvironment.value = false
  }
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

.form-tip {
  color: #7a8294;
  font-size: 12px;
  line-height: 20px;
  margin-top: 4px;
}
</style>
