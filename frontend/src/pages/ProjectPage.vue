<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>项目管理</h1>
        <p>项目是产品的上层归属，一个项目可以绑定多个产品。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadProjects">刷新</el-button>
        <el-button type="primary" @click="openCreateDialog">新增项目</el-button>
      </div>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <div class="toolbar-left">
          <el-input v-model="keyword" placeholder="搜索项目名称、编码" clearable />
          <el-select v-model="statusFilter" placeholder="全部状态" clearable>
            <el-option label="启用" value="ACTIVE" />
            <el-option label="停用" value="DISABLED" />
          </el-select>
        </div>
      </div>
      <el-alert v-if="errorMessage" class="project-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="project-table">
        <el-table-column label="项目" min-width="240">
          <template #default="{ row }">
            <div class="project-main">
              <strong>{{ row.name }}</strong>
              <span>{{ row.code }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="产品数量" width="110" prop="productCount" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="row.status === 'DISABLED' ? 'info' : 'success'" effect="light">
              {{ projectStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="说明" min-width="260">
          <template #default="{ row }">{{ row.description || '-' }}</template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="180">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="128">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
              <el-tooltip content="绑定、解绑或换绑产品" placement="top">
                <el-button link type="primary" @click="openProductDialog(row)">产品</el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogMode === 'create' ? '新增项目' : '编辑项目'" width="520px" destroy-on-close>
      <el-form :model="form" label-width="90px">
        <el-form-item label="项目名称" required>
          <el-input v-model="form.name" placeholder="项目 A" />
        </el-form-item>
        <el-form-item label="项目编码">
          <el-input v-model="form.code" :disabled="dialogMode === 'edit'" placeholder="保存时自动生成" />
        </el-form-item>
        <el-form-item label="项目状态">
          <el-segmented v-model="form.status" :options="statusOptions" />
        </el-form-item>
        <el-form-item label="项目说明">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="项目范围或备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitProject">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="productDialogVisible" title="绑定/换绑产品" width="620px" destroy-on-close>
      <div v-if="selectedProject" class="binding-head">
        <strong>{{ selectedProject.name }}</strong>
        <span>产品只能归属一个项目。选择其他项目下的产品并保存后，会从原项目换绑到当前项目；取消已选产品会从当前项目解绑。</span>
      </div>
      <el-alert
        v-if="bindingNotice"
        class="project-alert"
        type="warning"
        :closable="false"
        :title="bindingNotice"
      />
      <el-alert
        v-if="productErrorMessage"
        class="project-alert"
        type="warning"
        :closable="false"
        :title="productErrorMessage"
      />
      <el-select
        v-model="selectedProductIds"
        class="product-select"
        multiple
        filterable
        clearable
        :loading="productLoading"
        placeholder="选择一个或多个产品"
      >
        <el-option
          v-for="item in bindableProducts"
          :key="item.id"
          :label="productOptionLabel(item)"
          :value="item.id"
        />
      </el-select>
      <template #footer>
        <el-button @click="productDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="bindingSubmitting" @click="submitProductBinding">保存绑定</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listEnvironments, updateEnvironment, type EnvironmentInfo } from '@/api/environments'
import { createProject, listProjects, updateProject, type ProjectInfo, type ProjectPayload } from '@/api/projects'

const keyword = ref('')
const statusFilter = ref('')
const loading = ref(false)
const submitting = ref(false)
const productLoading = ref(false)
const bindingSubmitting = ref(false)
const errorMessage = ref('')
const productErrorMessage = ref('')
const projects = ref<ProjectInfo[]>([])
const products = ref<EnvironmentInfo[]>([])
const dialogVisible = ref(false)
const productDialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const selectedProject = ref<ProjectInfo | null>(null)
const selectedProductIds = ref<string[]>([])
const form = ref<ProjectPayload>(emptyProjectForm())

const statusOptions = [
  { label: '启用', value: 'ACTIVE' },
  { label: '停用', value: 'DISABLED' },
]

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return projects.value.filter((item) => {
    const keywordMatched = !q || `${item.name} ${item.code} ${item.description}`.toLowerCase().includes(q)
    const statusMatched = !statusFilter.value || item.status === statusFilter.value
    return keywordMatched && statusMatched
  })
})

const bindableProducts = computed(() => {
  const projectId = selectedProject.value?.id ?? ''
  return products.value.filter((item) => item.productStatus !== 'DISABLED' || item.projectId === projectId)
})

const rebindingProducts = computed(() => {
  const projectId = selectedProject.value?.id ?? ''
  const selected = new Set(selectedProductIds.value)
  return products.value.filter((item) => selected.has(item.id) && item.projectId && item.projectId !== projectId)
})

const unbindingProducts = computed(() => {
  const projectId = selectedProject.value?.id ?? ''
  const selected = new Set(selectedProductIds.value)
  return products.value.filter((item) => item.projectId === projectId && !selected.has(item.id))
})

const bindingNotice = computed(() => {
  const notices: string[] = []
  if (rebindingProducts.value.length > 0) {
    notices.push(`${rebindingProducts.value.length} 个产品将从其他项目换绑到当前项目`)
  }
  if (unbindingProducts.value.length > 0) {
    notices.push(`${unbindingProducts.value.length} 个产品将从当前项目解绑，解绑后不再归属任何项目`)
  }
  return notices.join('；')
})

function emptyProjectForm(): ProjectPayload {
  return {
    id: '',
    name: '',
    code: '',
    description: '',
    status: 'ACTIVE',
  }
}

async function loadProjects() {
  loading.value = true
  errorMessage.value = ''
  try {
    projects.value = await listProjects()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '项目数据加载失败'
  } finally {
    loading.value = false
  }
}

async function loadProducts() {
  productLoading.value = true
  productErrorMessage.value = ''
  try {
    products.value = await listEnvironments()
  } catch (error) {
    productErrorMessage.value = error instanceof Error ? error.message : '产品数据加载失败'
  } finally {
    productLoading.value = false
  }
}

function openCreateDialog() {
  dialogMode.value = 'create'
  form.value = emptyProjectForm()
  dialogVisible.value = true
}

function openEditDialog(row: ProjectInfo) {
  dialogMode.value = 'edit'
  form.value = {
    id: row.id,
    name: row.name,
    code: row.code,
    description: row.description,
    status: row.status,
  }
  dialogVisible.value = true
}

async function openProductDialog(row: ProjectInfo) {
  selectedProject.value = row
  productDialogVisible.value = true
  await loadProducts()
  selectedProductIds.value = products.value.filter((item) => item.projectId === row.id).map((item) => item.id)
}

async function submitProject() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写项目名称')
    return
  }
  submitting.value = true
  try {
    const payload = trimProjectPayload(form.value)
    if (dialogMode.value === 'create') {
      await createProject(payload)
      ElMessage.success('项目已创建')
    } else {
      await updateProject(form.value.id, payload)
      ElMessage.success('项目已更新')
    }
    dialogVisible.value = false
    await loadProjects()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '项目保存失败')
  } finally {
    submitting.value = false
  }
}

async function submitProductBinding() {
  if (!selectedProject.value) return
  if (rebindingProducts.value.length > 0 || unbindingProducts.value.length > 0) {
    try {
      await ElMessageBox.confirm(bindingConfirmText(), '确认更新产品归属', {
        confirmButtonText: '确认保存',
        cancelButtonText: '取消',
        type: 'warning',
      })
    } catch {
      return
    }
  }
  bindingSubmitting.value = true
  try {
    const projectId = selectedProject.value.id
    const selected = new Set(selectedProductIds.value)
    const changes = products.value.filter((item) => {
      const shouldBind = selected.has(item.id)
      return (shouldBind && item.projectId !== projectId) || (!shouldBind && item.projectId === projectId)
    })
    await Promise.all(changes.map((item) => {
      const shouldBind = selected.has(item.id)
      return updateEnvironment(item.id, {
        projectId: shouldBind ? projectId : '',
        productStatus: shouldBind ? 'BOUND' : 'UNBOUND',
      })
    }))
    ElMessage.success('产品绑定已更新')
    productDialogVisible.value = false
    await Promise.all([loadProjects(), loadProducts()])
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '产品绑定失败')
  } finally {
    bindingSubmitting.value = false
  }
}

function productOptionLabel(item: EnvironmentInfo) {
  const currentProjectId = selectedProject.value?.id ?? ''
  if (!item.projectId) return `${item.name}（未绑定项目）`
  if (item.projectId === currentProjectId) return `${item.name}（当前项目）`
  return `${item.name}（已绑定：${item.projectName || item.projectId}，选中后将换绑）`
}

function bindingConfirmText() {
  const items: string[] = []
  if (rebindingProducts.value.length > 0) {
    items.push(`${rebindingProducts.value.length} 个产品会从原项目换绑到 ${selectedProject.value?.name ?? '当前项目'}`)
  }
  if (unbindingProducts.value.length > 0) {
    items.push(`${unbindingProducts.value.length} 个产品会从 ${selectedProject.value?.name ?? '当前项目'} 解绑，变为未绑定项目`)
  }
  return items.join('；') || '确认更新产品归属？'
}

function trimProjectPayload(payload: ProjectPayload): ProjectPayload {
  const code = normalizeProjectCode(payload.code) || generateProjectCode(payload.name)
  return {
    id: payload.id.trim() || `proj-${code}`,
    name: payload.name.trim(),
    code,
    description: payload.description.trim(),
    status: payload.status === 'DISABLED' ? 'DISABLED' : 'ACTIVE',
  }
}

function normalizeProjectCode(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function generateProjectCode(name: string) {
  if (/[^\x00-\x7F]/.test(name)) return `project-${timestampCode()}`
  return normalizeProjectCode(name) || `project-${timestampCode()}`
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

function projectStatusLabel(status: string) {
  return status === 'DISABLED' ? '停用' : '启用'
}

function formatTime(value: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  const pad = (item: number) => String(item).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

watch(
  () => form.value.name,
  (name) => {
    if (dialogMode.value === 'create') {
      form.value.code = name.trim() ? generateProjectCode(name) : ''
    }
  },
)

onMounted(loadProjects)
</script>

<style scoped>
.head-actions {
  display: flex;
  gap: 10px;
}

.project-alert {
  margin-bottom: 12px;
}

.project-table :deep(.cell) {
  overflow-wrap: anywhere;
}

.project-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.project-main strong {
  color: #2f3847;
  font-size: 14px;
}

.project-main span {
  color: #7a8294;
  font-size: 12px;
}

.table-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  white-space: nowrap;
}

.table-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.binding-head {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.binding-head span {
  color: #6f7787;
  font-size: 13px;
}

.product-select {
  width: 100%;
}
</style>
