<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境管理</h1>
        <p>维护环境与基础资源的关联范围：K8s namespace、Harbor project、Jenkins view。</p>
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
        title="V1 主线：基础资源在“基础资源”菜单维护；环境只关联资源与作用域；.secrets 只用于研发阶段启动配置。"
      />
      <el-alert
        v-if="blockedProjectEnvironmentCount > 0"
        type="warning"
        :closable="false"
        :title="`${blockedProjectEnvironmentCount} 个项目环境 Agent 未就绪，远程发布/部署提交前会被阻断。`"
      />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="keyword" placeholder="搜索环境、编码" clearable />
          <el-select v-model="networkMode" placeholder="全部网络模式" clearable>
            <el-option label="平台直连" value="DIRECT" />
            <el-option label="Agent 模式" value="AGENT" />
          </el-select>
        </div>
      </div>
      <el-alert v-if="errorMessage" class="environment-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="name" label="环境" min-width="150" />
        <el-table-column prop="code" label="编码" min-width="150" />
        <el-table-column label="K8s / namespace" min-width="210">
          <template #default="{ row }">{{ resourceName(kubernetesClusters, row.clusterId) }} / {{ row.namespace || '-' }}</template>
        </el-table-column>
        <el-table-column label="Harbor / project" min-width="210">
          <template #default="{ row }">{{ resourceName(harborRegistries, row.registryId) }} / {{ row.registryProject || '-' }}</template>
        </el-table-column>
        <el-table-column label="Jenkins / view" min-width="210">
          <template #default="{ row }">{{ resourceName(jenkinsInstances, row.jenkinsId) }} / {{ row.jenkinsView || '-' }}</template>
        </el-table-column>
        <el-table-column label="网络" min-width="110">
          <template #default="{ row }">{{ row.networkMode === 'DIRECT' ? '平台直连' : 'Agent 模式' }}</template>
        </el-table-column>
        <el-table-column label="Agent" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.agentStatus" /></template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="170">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="primary" @click="openDrawer(row)">连接配置</el-button>
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
        <el-form-item label="环境编码" required>
          <el-input v-model="form.code" placeholder="project-x-prod" />
          <div class="form-tip">保存后系统生成环境 ID：env-环境编码</div>
        </el-form-item>
        <el-form-item label="环境类型" required><el-segmented v-model="form.type" :options="typeOptions" /></el-form-item>
        <el-form-item label="网络模式" required><el-segmented v-model="form.networkMode" :options="networkOptions" /></el-form-item>
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
import { computed, onMounted, ref } from 'vue'
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
const networkMode = ref('')
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
  { label: '项目环境', value: 'PROJECT' },
  { label: '本地环境', value: 'LOCAL' },
]

const networkOptions = [
  { label: 'Agent 模式', value: 'AGENT' },
  { label: '平台直连', value: 'DIRECT' },
]

const blockedProjectEnvironmentCount = computed(
  () => environments.value.filter((item) => item.type === 'PROJECT' && item.networkMode === 'AGENT' && item.agentStatus !== 'ONLINE').length,
)

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return environments.value.filter((item) => {
    const keywordMatched = !q || `${item.name} ${item.code}`.toLowerCase().includes(q)
    const modeMatched = !networkMode.value || item.networkMode === networkMode.value
    return keywordMatched && modeMatched
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
    networkMode: 'AGENT',
    clusterId: '',
    namespace: '',
    registryId: '',
    registryProject: '',
    jenkinsId: '',
    jenkinsView: '',
  }
}

function resourceName(items: IntegrationResource[], id: string) {
  if (!id) return '-'
  return items.find((item) => item.id === id)?.name || id
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
  form.value = {
    ...emptyEnvironmentForm(),
    clusterId: kubernetesClusters.value[0]?.id || '',
    registryId: harborRegistries.value[0]?.id || '',
    jenkinsId: jenkinsInstances.value[0]?.id || '',
  }
  dialogVisible.value = true
}

function openEditDialog(row: EnvironmentInfo) {
  dialogMode.value = 'edit'
  form.value = { ...row }
  dialogVisible.value = true
}

async function submitEnvironment() {
  if (!form.value.name.trim() || !form.value.code.trim()) {
    ElMessage.warning('请完整填写环境名称和编码')
    return
  }
  if (!form.value.clusterId || !form.value.namespace.trim() || !form.value.registryId || !form.value.registryProject.trim()) {
    ElMessage.warning('请完整选择 K8s/Harbor 资源并填写 namespace 与 project')
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
  return {
    ...payload,
    id: payload.id.trim(),
    name: payload.name.trim(),
    code: payload.code.trim(),
    clusterId: payload.clusterId.trim(),
    namespace: payload.namespace.trim(),
    registryId: payload.registryId.trim(),
    registryProject: payload.registryProject.trim(),
    jenkinsId: payload.jenkinsId.trim(),
    jenkinsView: payload.jenkinsView.trim(),
  }
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
