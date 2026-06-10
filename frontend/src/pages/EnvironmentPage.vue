<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境管理</h1>
        <p>维护真实环境主数据，完成环境创建、编辑和连通性校验，再接入 agent 与后续发布流程。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadEnvironments">刷新状态</el-button>
        <el-button type="primary" @click="openCreateDialog">新增环境</el-button>
      </div>
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 顺序：先维护真实环境，再校验 agent 绑定，最后继续发布、部署与运行态能力联调。"
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
        <el-button>批量连接测试</el-button>
      </div>
      <el-alert v-if="errorMessage" class="environment-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="name" label="环境" min-width="160" />
        <el-table-column prop="code" label="编码" min-width="160" />
        <el-table-column label="类型" min-width="110">
          <template #default="{ row }">{{ row.type === 'LOCAL' ? '本地环境' : '项目环境' }}</template>
        </el-table-column>
        <el-table-column label="网络模式" min-width="120">
          <template #default="{ row }">{{ row.networkMode === 'DIRECT' ? '平台直连' : 'Agent 模式' }}</template>
        </el-table-column>
        <el-table-column label="Agent" min-width="110">
          <template #default="{ row }"><StatusTag :status="row.agentStatus" /></template>
        </el-table-column>
        <el-table-column prop="lastCheckAt" label="最近测试" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.lastCheckAt) }}</template>
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

    <EnvironmentConfigDrawer
      v-model:visible="drawerVisible"
      :environment="activeEnvironment"
      :checking="checkingEnvironment"
      @check="handleCheckEnvironment"
    />

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新增环境' : '编辑环境'"
      width="520px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" label-width="96px">
        <el-form-item label="环境 ID" required>
          <el-input v-model="form.id" :disabled="dialogMode === 'edit'" placeholder="env-project-x-prod" />
        </el-form-item>
        <el-form-item label="环境名称" required>
          <el-input v-model="form.name" placeholder="项目 X 生产" />
        </el-form-item>
        <el-form-item label="环境编码" required>
          <el-input v-model="form.code" placeholder="project-x-prod" />
        </el-form-item>
        <el-form-item label="环境类型" required>
          <el-segmented v-model="form.type" :options="typeOptions" />
        </el-form-item>
        <el-form-item label="网络模式" required>
          <el-segmented v-model="form.networkMode" :options="networkOptions" />
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
import {
  checkEnvironment,
  createEnvironment,
  listEnvironments,
  updateEnvironment,
  type EnvironmentInfo,
  type EnvironmentPayload,
} from '@/api/environments'
import { formatDateTime } from '@/utils/format'

const keyword = ref('')
const networkMode = ref('')
const drawerVisible = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const activeEnvironment = ref<EnvironmentInfo | null>(null)
const environments = ref<EnvironmentInfo[]>([])
const loading = ref(false)
const submitting = ref(false)
const checkingEnvironment = ref(false)
const errorMessage = ref('')
const form = ref<EnvironmentPayload>({
  id: '',
  name: '',
  code: '',
  type: 'PROJECT',
  networkMode: 'AGENT',
})

const typeOptions = [
  { label: '项目环境', value: 'PROJECT' },
  { label: '本地环境', value: 'LOCAL' },
]

const networkOptions = [
  { label: 'Agent 模式', value: 'AGENT' },
  { label: '平台直连', value: 'DIRECT' },
]

const blockedProjectEnvironmentCount = computed(
  () =>
    environments.value.filter(
      (item) => item.type === 'PROJECT' && item.networkMode === 'AGENT' && item.agentStatus !== 'ONLINE',
    ).length,
)

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return environments.value.filter((item) => {
    const keywordMatched = !q || `${item.name} ${item.code}`.toLowerCase().includes(q)
    const modeMatched = !networkMode.value || item.networkMode === networkMode.value
    return keywordMatched && modeMatched
  })
})

async function loadEnvironments() {
  loading.value = true
  errorMessage.value = ''
  try {
    environments.value = await listEnvironments()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '环境列表加载失败'
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
    id: '',
    name: '',
    code: '',
    type: 'PROJECT',
    networkMode: 'AGENT',
  }
  dialogVisible.value = true
}

function openEditDialog(row: EnvironmentInfo) {
  dialogMode.value = 'edit'
  form.value = {
    id: row.id,
    name: row.name,
    code: row.code,
    type: row.type,
    networkMode: row.networkMode,
    status: row.status,
  }
  dialogVisible.value = true
}

async function submitEnvironment() {
  if (!form.value.id.trim() || !form.value.name.trim() || !form.value.code.trim()) {
    ElMessage.warning('请完整填写环境 ID、名称和编码')
    return
  }

  submitting.value = true
  try {
    if (dialogMode.value === 'create') {
      await createEnvironment({
        ...form.value,
        id: form.value.id.trim(),
        name: form.value.name.trim(),
        code: form.value.code.trim(),
      })
      ElMessage.success('环境已创建')
    } else {
      await updateEnvironment(form.value.id, {
        name: form.value.name.trim(),
        code: form.value.code.trim(),
        type: form.value.type,
        networkMode: form.value.networkMode,
        status: form.value.status,
      })
      ElMessage.success('环境已更新')
    }
    dialogVisible.value = false
    await loadEnvironments()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '环境保存失败')
  } finally {
    submitting.value = false
  }
}

async function handleCheckEnvironment(id: string) {
  checkingEnvironment.value = true
  try {
    const result = await checkEnvironment(id)
    ElMessage.success(`连接测试完成：${result.status}`)
    await loadEnvironments()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '连接测试失败')
  } finally {
    checkingEnvironment.value = false
  }
}

onMounted(loadEnvironments)
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
</style>
