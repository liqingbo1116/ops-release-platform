<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>Agent 管理</h1>
        <p>项目环境 Agent 主动上报心跳、资源状态与任务执行进度，平台侧通过刷新查看最新状态。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadAgents">刷新状态</el-button>
        <el-button type="primary" @click="drawerVisible = true">注册 Agent</el-button>
      </div>
    </div>

    <div class="metric-grid six">
      <MetricCard label="注册 Agent" :value="agents.length" foot="平台已接入" />
      <MetricCard label="在线" :value="onlineCount" foot="心跳正常" tone="good" />
      <MetricCard label="待绑定" :value="pendingClaimCount" foot="需绑定远程产品" tone="warn" />
      <MetricCard label="执行中" :value="runningCount" foot="发布 / 部署" />
      <MetricCard label="离线" :value="offlineCount" foot="需排查" tone="bad" />
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 Agent 研发阶段按二进制直接启动；真实联调前需确认 Agent 可访问平台 API、项目环境 Kubernetes 与 Harbor。Jenkins 由平台后端直连本地资源。"
      />
      <el-alert
        v-if="offlineCount > 0"
        type="warning"
        :closable="false"
        :title="`${offlineCount} 个 Agent 离线，对应项目环境的远程发布/部署会被阻断。`"
      />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索 Agent、环境、能力" clearable />
        <el-button :loading="loading" @click="loadData">刷新</el-button>
      </div>
      <el-alert v-if="errorMessage" class="agent-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="name" label="Agent" min-width="160" />
        <el-table-column label="环境" min-width="190">
          <template #default="{ row }">
            <span v-if="row.environmentName">{{ row.environmentName }}</span>
            <el-tag v-else type="warning" effect="plain">待绑定</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="version" label="版本" min-width="100" />
        <el-table-column prop="lastHeartbeatAt" label="心跳" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.lastHeartbeatAt) }}</template>
        </el-table-column>
        <el-table-column label="可执行能力" min-width="220">
          <template #default="{ row }">{{ joinCapabilities(row.capabilities) }}</template>
        </el-table-column>
        <el-table-column label="远程资源" min-width="280">
          <template #default="{ row }">
            <div class="runtime-resource">
              <el-tooltip
                effect="dark"
                placement="top"
                :content="runtimeTooltip(row, 'kubernetes')"
              >
                <div class="runtime-resource-row">
                  <span class="runtime-resource-name">K8s</span>
                  <StatusTag :status="runtimeComponent(row, 'kubernetes').status" />
                </div>
              </el-tooltip>
              <el-tooltip
                effect="dark"
                placement="top"
                :content="runtimeTooltip(row, 'harbor')"
              >
                <div class="runtime-resource-row">
                  <span class="runtime-resource-name">Harbor</span>
                  <StatusTag :status="runtimeComponent(row, 'harbor').status" />
                </div>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="130">
          <template #default="{ row }">
            <el-space>
              <StatusTag :status="row.status" />
              <el-tag v-if="row.claimStatus === 'PENDING_CLAIM'" type="warning" effect="plain">待绑定</el-tag>
            </el-space>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.claimStatus === 'PENDING_CLAIM'" link type="primary" @click="openClaim(row)">绑定产品</el-button>
            <span v-else class="muted-text">已绑定</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <AgentRegisterDrawer v-model:visible="drawerVisible" />
    <el-dialog v-model="claimDialogVisible" title="绑定 Agent 远程产品" width="420px">
      <el-form label-position="top">
        <el-form-item label="Agent">
          <el-input :model-value="claimAgentName" readonly />
        </el-form-item>
        <el-form-item label="远程产品">
          <el-select v-model="claimEnvironmentId" style="width: 100%" placeholder="选择远程产品">
            <el-option v-for="env in claimableEnvironments" :key="env.id" :label="env.name" :value="env.id" />
          </el-select>
          <div v-if="claimableEnvironments.length === 0" class="form-tip">暂无可绑定的远程产品，本地产品不需要绑定 Agent。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="claimDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="claiming" :disabled="!claimEnvironmentId" @click="submitClaim">确认绑定</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AgentRegisterDrawer from '@/components/AgentRegisterDrawer.vue'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { claimAgent, listAgents, type AgentInfo } from '@/api/agents'
import { listEnvironments, type EnvironmentInfo } from '@/api/environments'
import { formatDateTime, joinCapabilities } from '@/utils/format'

const keyword = ref('')
const drawerVisible = ref(false)
const agents = ref<AgentInfo[]>([])
const environments = ref<EnvironmentInfo[]>([])
const loading = ref(false)
const errorMessage = ref('')
const claimDialogVisible = ref(false)
const claimEnvironmentId = ref('')
const claimAgentId = ref('')
const claimAgentName = ref('')
const claiming = ref(false)

const onlineCount = computed(() => agents.value.filter((item) => item.status === 'ONLINE').length)
const offlineCount = computed(() => agents.value.filter((item) => item.status === 'OFFLINE').length)
const runningCount = computed(() => agents.value.filter((item) => item.currentTaskId).length)
const pendingClaimCount = computed(() => agents.value.filter((item) => item.claimStatus === 'PENDING_CLAIM').length)
const claimableEnvironments = computed(() => environments.value.filter((item) => item.type === 'PROJECT'))

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return agents.value
  return agents.value.filter((item) =>
    `${item.name} ${item.environmentName} ${item.capabilities.join(' ')}`.toLowerCase().includes(q),
  )
})

async function loadAgents() {
  return loadData()
}

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [agentItems, environmentItems] = await Promise.all([listAgents(), listEnvironments()])
    agents.value = agentItems
    environments.value = environmentItems
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Agent 状态加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function openClaim(row: AgentInfo) {
  claimAgentId.value = row.id
  claimAgentName.value = row.name || row.id
  claimEnvironmentId.value = ''
  claimDialogVisible.value = true
}

function runtimeComponent(row: AgentInfo, key: 'kubernetes' | 'harbor') {
  const component = row.runtimeStatus?.[key]
  if (!component?.status) {
    return {
      status: 'UNKNOWN',
      message: row.claimStatus === 'PENDING_CLAIM' ? '绑定产品后查看上报状态' : '等待 Agent 上报',
    }
  }
  const itemLabel = key === 'kubernetes' ? 'namespace' : 'project'
  const itemCount = component.items?.length ?? 0
  const suffix = component.status === 'HEALTHY' ? `，已上报 ${itemCount} 个 ${itemLabel}` : ''
  return {
    status: component.status,
    message: `${component.message || runtimeStatusText(component.status)}${suffix}`,
  }
}

function runtimeTooltip(row: AgentInfo, key: 'kubernetes' | 'harbor') {
  return runtimeComponent(row, key).message
}

function runtimeStatusText(status: string) {
  if (status === 'HEALTHY') return '连接正常'
  if (status === 'UNHEALTHY') return '连接异常'
  return '状态未知'
}

async function submitClaim() {
  if (!claimEnvironmentId.value || !claimAgentId.value) return
  claiming.value = true
  try {
    await claimAgent(claimAgentId.value, claimEnvironmentId.value)
    ElMessage.success('Agent 已绑定远程产品')
    claimDialogVisible.value = false
    await loadData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : 'Agent 绑定远程产品失败')
  } finally {
    claiming.value = false
  }
}
</script>

<style scoped>
.head-actions {
  display: flex;
  gap: 10px;
}

.agent-alert {
  margin-bottom: 12px;
}

.readiness-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.muted-text {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.form-tip {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.runtime-resource {
  display: grid;
  gap: 6px;
}

.runtime-resource-row {
  display: grid;
  grid-template-columns: 48px 52px;
  align-items: center;
  gap: 8px;
  min-height: 24px;
}

.runtime-resource-name {
  color: var(--el-text-color-regular);
  font-size: 13px;
}

</style>
