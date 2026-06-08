<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境基线列表</h1>
        <p>从真实运行态采集生成交付基线，替代难以维护的传统产品版本。</p>
      </div>
      <el-button type="primary" @click="openCreateDialog">从运行环境生成基线</el-button>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索基线 ID、来源环境、用途" clearable />
        <div class="top-actions">
          <el-button>导出清单</el-button>
          <el-button :disabled="!selectedRows.length" @click="handleBatchLock">批量锁定</el-button>
        </div>
      </div>
      <el-table v-loading="loading" :data="filteredRows" class="wide-table" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="id" label="基线 ID" min-width="170" />
        <el-table-column prop="name" label="基线名称" min-width="220" />
        <el-table-column prop="sourceEnvironmentName" label="来源环境" min-width="150" />
        <el-table-column prop="serviceCount" label="服务数" min-width="90" />
        <el-table-column prop="createdBy" label="创建人" min-width="90" />
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="purpose" label="用途" min-width="140" />
        <el-table-column label="操作" fixed="right" width="100">
          <template #default="{ row }">
            <el-button link type="primary" @click="$router.push(`/baselines/${row.id}`)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="createDialogVisible" title="从运行态生成基线" width="520px">
      <el-form label-position="top">
        <el-form-item label="来源环境">
          <el-select v-model="createForm.sourceEnvironmentId" placeholder="选择来源环境">
            <el-option
              v-for="environment in environments"
              :key="environment.id"
              :label="`${environment.name} / ${environment.code}`"
              :value="environment.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="基线名称">
          <el-input v-model="createForm.name" placeholder="例如 project-x-prod-20260608-2200" />
        </el-form-item>
        <el-form-item label="用途">
          <el-input v-model="createForm.purpose" placeholder="例如 项目 X 发布前快照" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreateBaseline">生成基线</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { createBaseline, listBaselines, lockBaseline } from '@/api/baselines'
import { listEnvironments } from '@/api/environments'
import StatusTag from '@/components/StatusTag.vue'
import { baselineMockData } from '@/api/mockData/baseline'
import { environmentMockData } from '@/api/mockData/environment'
import { formatDateTime } from '@/utils/format'

const router = useRouter()
const keyword = ref('')
const loading = ref(false)
const rows = ref([...baselineMockData.baselines])
const environments = ref([...environmentMockData.environments])
const selectedRows = ref<typeof baselineMockData.baselines>([])
const createDialogVisible = ref(false)
const creating = ref(false)
const createForm = ref({
  sourceEnvironmentId: '',
  name: '',
  purpose: '',
})

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((item) =>
    `${item.id} ${item.name} ${item.sourceEnvironmentName} ${item.purpose}`.toLowerCase().includes(q),
  )
})

function handleSelectionChange(selection: typeof baselineMockData.baselines) {
  selectedRows.value = selection
}

function openCreateDialog() {
  const defaultEnvironment = environments.value[0]
  createForm.value = {
    sourceEnvironmentId: defaultEnvironment?.id || '',
    name: defaultEnvironment ? `${defaultEnvironment.code}-${new Date().toISOString().slice(0, 16).replace(/[-:T]/g, '')}` : '',
    purpose: '远程部署前运行态基线',
  }
  createDialogVisible.value = true
}

async function handleCreateBaseline() {
  if (!createForm.value.sourceEnvironmentId || !createForm.value.name.trim()) {
    ElMessage.warning('请填写来源环境和基线名称')
    return
  }
  creating.value = true
  try {
    const detail = await createBaseline({
      sourceEnvironmentId: createForm.value.sourceEnvironmentId,
      name: createForm.value.name.trim(),
      purpose: createForm.value.purpose.trim(),
    })
    createDialogVisible.value = false
    ElMessage.success('已根据运行态生成基线')
    await loadRows()
    await router.push(`/baselines/${detail.id}`)
  } catch {
    ElMessage.error('生成基线失败')
  } finally {
    creating.value = false
  }
}

async function handleBatchLock() {
  const pendingRows = selectedRows.value.filter((item) => item.status !== 'LOCKED')
  if (!pendingRows.length) {
    ElMessage.info('所选基线已经全部锁定')
    return
  }
  loading.value = true
  try {
    await Promise.all(pendingRows.map((item) => lockBaseline(item.id)))
    ElMessage.success(`已锁定 ${pendingRows.length} 个基线`)
    await loadRows()
  } catch {
    ElMessage.error('批量锁定失败')
  } finally {
    loading.value = false
  }
}

async function loadEnvironments() {
  try {
    environments.value = await listEnvironments()
  } catch {
    environments.value = [...environmentMockData.environments]
  }
}

async function loadRows() {
  loading.value = true
  try {
    rows.value = await listBaselines()
  } catch {
    ElMessage.warning('加载基线列表失败，已显示本地示例数据')
    rows.value = [...baselineMockData.baselines]
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadEnvironments()
  await loadRows()
})
</script>
