<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>基础资源</h1>
        <p>单独维护 K8s、Harbor、Jenkins 连接，并缓存 namespace、project、view 供环境关联。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadAll">刷新</el-button>
        <el-button type="primary" @click="openResourceCreateDialog">新增资源</el-button>
      </div>
    </div>

    <el-alert v-if="errorMessage" class="resource-alert" type="warning" :closable="false" :title="errorMessage" />

    <el-tabs v-model="resourceKind" class="resource-tabs">
      <el-tab-pane label="K8s 集群" name="kubernetes">
        <el-card shadow="never">
          <el-table v-loading="loading" :data="kubernetesClusters" class="wide-table">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column label="API Server" min-width="260">
              <template #default="{ row }">{{ resourceText(row.apiServer) }}</template>
            </el-table-column>
            <el-table-column label="命名空间" min-width="110">
              <template #default="{ row }">{{ row.namespaces.length }}</template>
            </el-table-column>
            <el-table-column label="状态" min-width="100">
              <template #default="{ row }"><StatusTag :status="row.status" /></template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatCheckTime(row.lastCheckAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" fixed="right" width="190">
              <template #default="{ row }">
                <el-button link type="primary" @click="openResourceEditDialog('kubernetes', row)">编辑</el-button>
                <el-button link type="primary" :loading="isResourceActionLoading('kubernetes', row.id, 'test')" @click="handleResourceAction('kubernetes', row.id, 'test')">测试</el-button>
                <el-button link type="primary" :loading="isResourceActionLoading('kubernetes', row.id, 'refresh')" @click="handleResourceAction('kubernetes', row.id, 'refresh')">刷新</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Harbor 仓库" name="harbor">
        <el-card shadow="never">
          <el-table v-loading="loading" :data="harborRegistries" class="wide-table">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="url" label="地址" min-width="240" />
            <el-table-column prop="username" label="用户名" min-width="120" />
            <el-table-column label="项目数" min-width="100">
              <template #default="{ row }">{{ row.projects.length }}</template>
            </el-table-column>
            <el-table-column label="状态" min-width="100">
              <template #default="{ row }"><StatusTag :status="row.status" /></template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatCheckTime(row.lastCheckAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" fixed="right" width="190">
              <template #default="{ row }">
                <el-button link type="primary" @click="openResourceEditDialog('harbor', row)">编辑</el-button>
                <el-button link type="primary" :loading="isResourceActionLoading('harbor', row.id, 'test')" @click="handleResourceAction('harbor', row.id, 'test')">测试</el-button>
                <el-button link type="primary" :loading="isResourceActionLoading('harbor', row.id, 'refresh')" @click="handleResourceAction('harbor', row.id, 'refresh')">刷新</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Jenkins" name="jenkins">
        <el-card shadow="never">
          <el-table v-loading="loading" :data="jenkinsInstances" class="wide-table">
            <el-table-column prop="name" label="名称" min-width="150" />
            <el-table-column prop="url" label="地址" min-width="240" />
            <el-table-column prop="username" label="用户名" min-width="120" />
            <el-table-column label="视图 / Job" min-width="110">
              <template #default="{ row }">{{ row.views.length }} / {{ row.jobs.length }}</template>
            </el-table-column>
            <el-table-column label="状态" min-width="100">
              <template #default="{ row }"><StatusTag :status="row.status" /></template>
            </el-table-column>
            <el-table-column label="最近检查" min-width="170">
              <template #default="{ row }">{{ formatCheckTime(row.lastCheckAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" fixed="right" width="190">
              <template #default="{ row }">
                <el-button link type="primary" @click="openResourceEditDialog('jenkins', row)">编辑</el-button>
                <el-button link type="primary" :loading="isResourceActionLoading('jenkins', row.id, 'test')" @click="handleResourceAction('jenkins', row.id, 'test')">测试</el-button>
                <el-button link type="primary" :loading="isResourceActionLoading('jenkins', row.id, 'refresh')" @click="handleResourceAction('jenkins', row.id, 'refresh')">刷新</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="resourceDialogVisible" :title="resourceDialogTitle" width="620px" destroy-on-close>
      <el-form :model="resourceForm" label-width="120px">
        <el-form-item label="名称" required><el-input v-model="resourceForm.name" /></el-form-item>

        <template v-if="resourceKind === 'kubernetes'">
          <el-form-item label="Kubeconfig" :required="resourceDialogMode === 'create'">
            <el-input v-model="resourceForm.kubeconfig" type="textarea" :rows="10" placeholder="粘贴 kubeconfig；编辑时留空表示保留原值" />
          </el-form-item>
        </template>

        <template v-else-if="resourceKind === 'harbor'">
          <el-form-item label="地址" required>
            <el-input v-model="resourceForm.url" placeholder="https://reg.example.com:5000；未写协议时默认 https" />
          </el-form-item>
          <el-form-item label="用户名" required><el-input v-model="resourceForm.username" /></el-form-item>
          <el-form-item label="密码" :required="resourceDialogMode === 'create'"><el-input v-model="resourceForm.password" type="password" show-password placeholder="编辑时留空表示保留原值" /></el-form-item>
          <el-form-item label="跳过 TLS 校验"><el-switch v-model="resourceForm.insecureSkipTLSVerify" /></el-form-item>
        </template>

        <template v-else>
          <el-form-item label="地址" required>
            <el-input v-model="resourceForm.url" placeholder="https://jenkins.example.com:8080；未写协议时默认 https" />
          </el-form-item>
          <el-form-item label="用户名" required><el-input v-model="resourceForm.username" /></el-form-item>
          <el-form-item label="密码" :required="resourceDialogMode === 'create'"><el-input v-model="resourceForm.token" type="password" show-password placeholder="编辑时留空表示保留原值" /></el-form-item>
          <el-form-item label="跳过 TLS 校验"><el-switch v-model="resourceForm.insecureSkipTLSVerify" /></el-form-item>
        </template>
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
import StatusTag from '@/components/StatusTag.vue'
import { formatDateTime } from '@/utils/format'
import {
  createHarborRegistry,
  createJenkinsInstance,
  createKubernetesCluster,
  listHarborRegistries,
  listJenkinsInstances,
  listKubernetesClusters,
  refreshHarborRegistry,
  refreshJenkinsInstance,
  refreshKubernetesCluster,
  testHarborRegistry,
  testJenkinsInstance,
  testKubernetesCluster,
  updateHarborRegistry,
  updateJenkinsInstance,
  updateKubernetesCluster,
  type HarborRegistry,
  type IntegrationResource,
  type IntegrationResourceKind,
  type JenkinsInstance,
  type KubernetesCluster,
} from '@/api/integrationResources'

type ResourceAction = 'test' | 'refresh'

type ResourceForm = {
  id: string
  name: string
  kubeconfig: string
  url: string
  username: string
  password: string
  token: string
  insecureSkipTLSVerify: boolean
}

const resourceKind = ref<IntegrationResourceKind>('kubernetes')
const kubernetesClusters = ref<KubernetesCluster[]>([])
const harborRegistries = ref<HarborRegistry[]>([])
const jenkinsInstances = ref<JenkinsInstance[]>([])
const loading = ref(false)
const errorMessage = ref('')
const resourceDialogVisible = ref(false)
const resourceDialogMode = ref<'create' | 'edit'>('create')
const resourceSubmitting = ref(false)
const resourceActionLoading = ref('')
const resourceForm = ref<ResourceForm>(emptyResourceForm())

const resourceDialogTitle = computed(() => {
  const prefix = resourceDialogMode.value === 'create' ? '新增' : '编辑'
  return `${prefix}${resourceKindName(resourceKind.value)}`
})

function emptyResourceForm(): ResourceForm {
  return {
    id: '',
    name: '',
    kubeconfig: '',
    url: '',
    username: '',
    password: '',
    token: '',
    insecureSkipTLSVerify: false,
  }
}

function resourceKindName(kind: IntegrationResourceKind) {
  return kind === 'kubernetes' ? 'K8s 集群' : kind === 'harbor' ? 'Harbor 仓库' : 'Jenkins'
}

function formatCheckTime(value: string) {
  return value ? formatDateTime(value) : '-'
}

function resourceText(value: string) {
  return value?.trim() || '-'
}

function resourceActionKey(kind: IntegrationResourceKind, id: string, action: ResourceAction) {
  return `${kind}:${id}:${action}`
}

function isResourceActionLoading(kind: IntegrationResourceKind, id: string, action: ResourceAction) {
  return resourceActionLoading.value === resourceActionKey(kind, id, action)
}

function generateResourceId(kind: IntegrationResourceKind, name: string) {
  const prefix = kind === 'kubernetes' ? 'k8s' : kind
  const slug = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return `${prefix}-${slug || Date.now()}`
}

function normalizeInputURL(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return ''
  return /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`
}

function schemeFromURL(value: string): 'http' | 'https' {
  return value.trim().toLowerCase().startsWith('http://') ? 'http' : 'https'
}

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [clusterItems, registryItems, jenkinsItems] = await Promise.all([
      listKubernetesClusters(),
      listHarborRegistries(),
      listJenkinsInstances(),
    ])
    kubernetesClusters.value = clusterItems
    harborRegistries.value = registryItems
    jenkinsInstances.value = jenkinsItems
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '基础资源数据加载失败'
  } finally {
    loading.value = false
  }
}

function openResourceCreateDialog() {
  resourceDialogMode.value = 'create'
  resourceForm.value = emptyResourceForm()
  resourceDialogVisible.value = true
}

function openResourceEditDialog(kind: IntegrationResourceKind, row: IntegrationResource) {
  resourceKind.value = kind
  resourceDialogMode.value = 'edit'
  const next = emptyResourceForm()
  next.id = row.id
  next.name = row.name
  if (kind === 'harbor') {
    const registry = row as HarborRegistry
    next.url = registry.url
    next.username = registry.username
    next.insecureSkipTLSVerify = registry.insecureSkipTLSVerify
  } else if (kind === 'jenkins') {
    const instance = row as JenkinsInstance
    next.url = instance.url
    next.username = instance.username
    next.insecureSkipTLSVerify = instance.insecureSkipTLSVerify
  }
  resourceForm.value = next
  resourceDialogVisible.value = true
}

async function submitResource() {
  if (!resourceForm.value.name.trim()) {
    ElMessage.warning('请填写资源名称')
    return
  }
  if (resourceKind.value === 'kubernetes' && resourceDialogMode.value === 'create' && !resourceForm.value.kubeconfig.trim()) {
    ElMessage.warning('请填写 kubeconfig')
    return
  }
  if (resourceKind.value === 'harbor' && (!resourceForm.value.url.trim() || !resourceForm.value.username.trim())) {
    ElMessage.warning('请完整填写 Harbor 地址和用户名')
    return
  }
  if (resourceKind.value === 'jenkins' && (!resourceForm.value.url.trim() || !resourceForm.value.username.trim())) {
    ElMessage.warning('请完整填写 Jenkins 地址和用户名')
    return
  }

  resourceSubmitting.value = true
  const isCreate = resourceDialogMode.value === 'create'
  const kind = resourceKind.value
  try {
    const id = isCreate ? generateResourceId(kind, resourceForm.value.name) : resourceForm.value.id
    if (kind === 'kubernetes') {
      const payload = {
        id,
        name: resourceForm.value.name.trim(),
        apiServer: '',
        context: undefined,
        kubeconfig: resourceForm.value.kubeconfig.trim() || undefined,
      }
      isCreate ? await createKubernetesCluster(payload) : await updateKubernetesCluster(payload.id, payload)
    } else if (kind === 'harbor') {
      const url = normalizeInputURL(resourceForm.value.url)
      const payload = {
        id,
        name: resourceForm.value.name.trim(),
        url,
        scheme: schemeFromURL(url),
        username: resourceForm.value.username.trim(),
        password: resourceForm.value.password.trim() || undefined,
        insecureSkipTLSVerify: resourceForm.value.insecureSkipTLSVerify,
      }
      isCreate ? await createHarborRegistry(payload) : await updateHarborRegistry(payload.id, payload)
    } else {
      const payload = {
        id,
        name: resourceForm.value.name.trim(),
        url: normalizeInputURL(resourceForm.value.url),
        username: resourceForm.value.username.trim(),
        token: resourceForm.value.token.trim() || undefined,
        insecureSkipTLSVerify: resourceForm.value.insecureSkipTLSVerify,
      }
      isCreate ? await createJenkinsInstance(payload) : await updateJenkinsInstance(payload.id, payload)
    }
    if (isCreate) {
      try {
        await refreshResource(kind, id)
        ElMessage.success('资源已保存，连接刷新完成')
      } catch (error) {
        ElMessage.warning(error instanceof Error ? `资源已保存，连接刷新失败：${error.message}` : '资源已保存，连接刷新失败')
      }
    } else {
      ElMessage.success('资源已保存')
    }
    resourceDialogVisible.value = false
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源保存失败')
  } finally {
    resourceSubmitting.value = false
  }
}

async function refreshResource(kind: IntegrationResourceKind, id: string) {
  if (kind === 'kubernetes') {
    await refreshKubernetesCluster(id)
  } else if (kind === 'harbor') {
    await refreshHarborRegistry(id)
  } else {
    await refreshJenkinsInstance(id)
  }
}

async function handleResourceAction(kind: IntegrationResourceKind, id: string, action: ResourceAction) {
  resourceActionLoading.value = resourceActionKey(kind, id, action)
  try {
    if (kind === 'kubernetes') {
      action === 'test' ? await testKubernetesCluster(id) : await refreshKubernetesCluster(id)
    } else if (kind === 'harbor') {
      action === 'test' ? await testHarborRegistry(id) : await refreshHarborRegistry(id)
    } else {
      action === 'test' ? await testJenkinsInstance(id) : await refreshJenkinsInstance(id)
    }
    ElMessage.success(action === 'test' ? '连接测试完成' : '刷新探测完成')
    await loadAll()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源探测失败')
    await loadAll()
  } finally {
    resourceActionLoading.value = ''
  }
}

onMounted(loadAll)
</script>

<style scoped>
.head-actions {
  display: flex;
  gap: 10px;
}

.resource-alert {
  margin-bottom: 12px;
}

.resource-tabs {
  margin-top: 4px;
}
</style>
