<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>环境管理</h1>
        <p>管理网络模式、K8s、Harbor、Nacos、MySQL、MinIO、凭证与连接测试。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadEnvironments">刷新状态</el-button>
        <el-button type="primary">新增环境</el-button>
      </div>
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 当前先基于 mock 验证远程项目环境发布/部署；真实联调前需要准备 Agent Linux 主机、docker compose、Jenkins、Harbor/Registry 与 Kubernetes。"
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
        <el-table-column label="操作" fixed="right" width="120">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDrawer(row)">连接配置</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <EnvironmentConfigDrawer v-model:visible="drawerVisible" :environment="activeEnvironment" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import EnvironmentConfigDrawer from '@/components/EnvironmentConfigDrawer.vue'
import StatusTag from '@/components/StatusTag.vue'
import { listEnvironments, type EnvironmentInfo } from '@/api/environments'
import { formatDateTime } from '@/utils/format'

const keyword = ref('')
const networkMode = ref('')
const drawerVisible = ref(false)
const activeEnvironment = ref<EnvironmentInfo | null>(null)
const environments = ref<EnvironmentInfo[]>([])
const loading = ref(false)
const errorMessage = ref('')

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
