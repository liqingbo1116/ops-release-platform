<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境管理</h1>
        <p>先维护 K8s、Harbor、Jenkins 资源，再让环境关联资源并填写 namespace、Harbor project、Jenkins view。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadAll">刷新</el-button>
        <el-button v-if="activeTab === 'environments'" type="primary" @click="openCreateDialog">新增环境</el-button>
        <el-button v-else type="primary" @click="openResourceCreateDialog">新增资源</el-button>
      </div>
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 主线：资源连接信息由平台维护；.secrets 只用于研发阶段启动服务，不作为正式环境主数据来源。"
      />
      <el-alert
        v-if="blockedProjectEnvironmentCount > 0"
        type="warning"
        :closable="false"
        :title="`${blockedProjectEnvironmentCount} 个项目环境 Agent 未就绪，远程发布/部署提交前会被阻断。`"
      />
    </div>

    <el-tabs v-model="activeTab" class="environment-tabs">
      <el-tab-pane label="环境" name="environments">
        <el-card shadow="never">
          <div class="toolbar">
            <div class="toolbar-left">
              <el-input v-model="keyword" placeholder="搜索环境、编码" clearable />
              <el-select v-model="networkMode" placeholder="全部网络模式" clearable>
                <el-option label="平台直连" value="DIRECT" />
                <el-option label="Agent 模式" value="AGENT" />
              </el-select>
            </div>
            <el-button>批量连接测试</el-button>
          </div>
          <el-alert v-if="errorMessage" class="environment-alert" type="warning" :closable="false" :title="errorMessage" />
          <el-table v-loading="loading" :data="filteredRows" class="wide-table">
            <el-table-column prop="name" label="环境" min-width="150" />
            <el-table-column prop="code" label="编码" min-width="150" />
            <el-table-column label="K8s / namespace" min-width="180">
              <template #default="{ row }">{{ resourceName(kubernetesClusters, row.clusterId) }} / {{ row.namespace || '-' }}</template>
            </el-table-column>
            <el-table-column label="Harbor / project" min-width="180">
              <template #default="{ row }">{{ resourceName(harborRegistries, row.registryId) }} / {{ row.registryProject || '-' }}</template>
            </el-table-column>
            <el-table-column label="Jenkins / view" min-width="180">
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
            <el-table-column label="操作" fixed="right" width="180">
              <template #default="{ row }">
                <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
                <el-button link type="primary" @click="openDrawer(row)">连接配置</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="K8s 集群" name="kubernetes">
        <el-card shadow="never">
          <el-table v-loading="loading" :data="kubernetesClusters" class="wide-table">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="id" label="资源 ID" min-width="150" />
            <el-table-column prop="apiServer" label="API Server" min-width="240" />
            <el-table-column prop="credentialRef" label="凭据引用" min-width="180" />
            <el-table-column label="状态" min-width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
            <el-table-column label="操作" fixed="right" width="100"><template #default="{ row }"><el-button link type="primary" @click="openResourceEditDialog('kubernetes', row)">编辑</el-button></template></el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Harbor 仓库" name="harbor">
        <el-card shadow="never">
          <el-table v-loading="loading" :data="harborRegistries" class="wide-table">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="id" label="资源 ID" min-width="150" />
            <el-table-column prop="url" label="地址" min-width="240" />
            <el-table-column prop="credentialRef" label="凭据引用" min-width="180" />
            <el-table-column label="状态" min-width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
            <el-table-column label="操作" fixed="right" width="100"><template #default="{ row }"><el-button link type="primary" @click="openResourceEditDialog('harbor', row)">编辑</el-button></template></el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Jenkins" name="jenkins">
        <el-card shadow="never">
          <el-table v-loading="loading" :data="jenkinsInstances" class="wide-table">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="id" label="资源 ID" min-width="150" />
            <el-table-column prop="url" label="地址" min-width="240" />
            <el-table-column prop="credentialRef" label="凭据引用" min-width="180" />
            <el-table-column label="状态" min-width="100"><template #default="{ row }"><StatusTag :status="row.status" /></template></el-table-column>
            <el-table-column label="操作" fixed="right" width="100"><template #default="{ row }"><el-button link type="primary" @click="openResourceEditDialog('jenkins', row)">编辑</el-button></template></el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <EnvironmentConfigDrawer
      v-model:visible="drawerVisible"
      :environment="activeEnvironment"
      :checking="checkingEnvironment"
      @check="handleCheckEnvironment"
    />

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增环境' : '编辑环境'" width="560px" destroy-on-close>
      <el-form :model="form" label-width="112px">
        <el-form-item label="环境名称" required><el-input v-model="form.name" placeholder="项目 X 生产" /></el-form-item>
        <el-form-item label="环境编码" required>
          <el-input v-model="form.code" placeholder="project-x-prod" />
          <div class="form-tip">保存后系统生成环境 ID：env-环境编码</div>
        </el-form-item>
        <el-form-item label="环境类型" required><el-segmented v-model="form.type" :options="typeOptions" /></el-form-item>
        <el-form-item label="网络模式" required><el-segmented v-model="form.networkMode" :options="networkOptions" /></el-form-item>
        <el-form-item label="K8s 集群" required>
          <el-select v-model="form.clusterId" placeholder="先维护 K8s 集群资源">
            <el-option v-for="item in kubernetesClusters" :key="item.id" :label="`${item.name}（${item.id}）`" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="命名空间" required><el-input v-model="form.namespace" placeholder="project-x-prod" /></el-form-item>
        <el-form-item label="Harbor 仓库" required>
          <el-select v-model="form.registryId" placeholder="先维护 Harbor 仓库资源">
            <el-option v-for="item in harborRegistries" :key="item.id" :label="`${item.name}（${item.id}）`" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Harbor project" required><el-input v-model="form.registryProject" placeholder="project-x" /></el-form-item>
        <el-form-item label="Jenkins">
          <el-select v-model="form.jenkinsId" placeholder="选择 Jenkins 资源" clearable>
            <el-option v-for="item in jenkinsInstances" :key="item.id" :label="`${item.name}（${item.id}）`" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Jenkins view"><el-input v-model="form.jenkinsView" placeholder="project-x" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitEnvironment">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resourceDialogVisible" :title="resourceDialogTitle" width="540px" destroy-on-close>
      <el-form :model="resourceForm" label-width="96px">
        <el-form-item label="资源 ID" required><el-input v-model="resourceForm.id" :disabled="resourceDialogMode === 'edit'" placeholder="例如 local-k3s / remote-harbor" /></el-form-item>
        <el-form-item label="名称" required><el-input v-model="resourceForm.name" /></el-form-item>
        <el-form-item :label="resourceAddressLabel" required><el-input v-model="resourceForm.address" /></el-form-item>
        <el-form-item label="凭据引用"><el-input v-model="resourceForm.credentialRef" placeholder="例如 secrets/local-harbor，不填写明文密码" /></el-form-item>
        <el-form-item label="状态"><el-select v-model="resourceForm.status"><el-option label="未知" value="UNKNOWN" /><el-option label="健康" value="HEALTHY" /><el-option label="异常" value="UNHEALTHY" /></el-select></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resourceDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="resourceSubmitting" @click="submitResource">保存</el-button>
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
  createHarborRegistry,
  createJenkinsInstance,
  createKubernetesCluster,
  listHarborRegistries,
  listJenkinsInstances,
  listKubernetesClusters,
  updateHarborRegistry,
  updateJenkinsInstance,
  updateKubernetesCluster,
  type HarborRegistry,
  type IntegrationResource,
  type IntegrationResourceKind,
  type JenkinsInstance,
  type KubernetesCluster,
} from '@/api/integrationResources'

type ActiveTab = 'environments' | IntegrationResourceKind

const activeTab = ref<ActiveTab>('environments')
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

const resourceDialogVisible = ref(false)
const resourceDialogMode = ref<'create' | 'edit'>('create')
const resourceKind = ref<IntegrationResourceKind>('kubernetes')
const resourceSubmitting = ref(false)
const resourceForm = ref({ id: '', name: '', address: '', credentialRef: '', status: 'UNKNOWN' })

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

const resourceDialogTitle = computed(() => {
  const prefix = resourceDialogMode.value === 'create' ? '新增' : '编辑'
  return `${prefix}${resourceKindName(resourceKind.value)}`
})

const resourceAddressLabel = computed(() => (resourceKind.value === 'kubernetes' ? 'API Server' : '地址'))

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

function resourceKindName(kind: IntegrationResourceKind) {
  return kind === 'kubernetes' ? 'K8s 集群' : kind === 'harbor' ? 'Harbor 仓库' : 'Jenkins'
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

function openResourceCreateDialog() {
  resourceKind.value = activeTab.value === 'environments' ? 'kubernetes' : activeTab.value
  resourceDialogMode.value = 'create'
  resourceForm.value = { id: '', name: '', address: '', credentialRef: '', status: 'UNKNOWN' }
  resourceDialogVisible.value = true
}

function openResourceEditDialog(kind: IntegrationResourceKind, row: IntegrationResource) {
  resourceKind.value = kind
  resourceDialogMode.value = 'edit'
  resourceForm.value = {
    id: row.id,
    name: row.name,
    address: kind === 'kubernetes' ? (row as KubernetesCluster).apiServer : (row as HarborRegistry | JenkinsInstance).url,
    credentialRef: row.credentialRef,
    status: row.status,
  }
  resourceDialogVisible.value = true
}

async function submitResource() {
  if (!resourceForm.value.id.trim() || !resourceForm.value.name.trim() || !resourceForm.value.address.trim()) {
    ElMessage.warning('请完整填写资源 ID、名称和地址')
    return
  }
  resourceSubmitting.value = true
  try {
    if (resourceKind.value === 'kubernetes') {
      const payload = { id: resourceForm.value.id.trim(), name: resourceForm.value.name.trim(), apiServer: resourceForm.value.address.trim(), credentialRef: resourceForm.value.credentialRef.trim(), status: resourceForm.value.status }
      resourceDialogMode.value === 'create' ? await createKubernetesCluster(payload) : await updateKubernetesCluster(payload.id, payload)
    } else if (resourceKind.value === 'harbor') {
      const payload = { id: resourceForm.value.id.trim(), name: resourceForm.value.name.trim(), url: resourceForm.value.address.trim(), credentialRef: resourceForm.value.credentialRef.trim(), status: resourceForm.value.status }
      resourceDialogMode.value === 'create' ? await createHarborRegistry(payload) : await updateHarborRegistry(payload.id, payload)
    } else {
      const payload = { id: resourceForm.value.id.trim(), name: resourceForm.value.name.trim(), url: resourceForm.value.address.trim(), credentialRef: resourceForm.value.credentialRef.trim(), status: resourceForm.value.status }
      resourceDialogMode.value === 'create' ? await createJenkinsInstance(payload) : await updateJenkinsInstance(payload.id, payload)
    }
    ElMessage.success('资源已保存')
    resourceDialogVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源保存失败')
  } finally {
    resourceSubmitting.value = false
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

.environment-tabs {
  margin-top: 4px;
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
