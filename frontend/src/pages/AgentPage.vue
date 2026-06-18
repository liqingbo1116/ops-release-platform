<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>Agent 管理</h1>
        <p>项目环境 Agent 主动拉取任务，执行镜像同步、kubectl、shell、健康检查并回传日志。</p>
      </div>
      <div class="head-actions">
        <el-button :loading="loading" @click="loadAgents">刷新状态</el-button>
        <el-button type="primary" @click="drawerVisible = true">注册 Agent</el-button>
      </div>
    </div>

    <div class="metric-grid six">
      <MetricCard label="注册 Agent" :value="agents.length" foot="绑定环境" />
      <MetricCard label="在线" :value="onlineCount" foot="心跳正常" tone="good" />
      <MetricCard label="执行中" :value="runningCount" foot="发布 / 部署" />
      <MetricCard label="离线" :value="offlineCount" foot="需排查" tone="bad" />
    </div>

    <div class="readiness-grid">
      <el-alert
        type="info"
        :closable="false"
        title="V1 Agent 研发阶段按二进制直接启动；真实联调前需确认 Agent 可访问平台 API、Jenkins、Harbor/Registry 与 Kubernetes。"
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
        <el-button :loading="loading" @click="loadData">下发探测任务</el-button>
      </div>
      <el-alert v-if="errorMessage" class="agent-alert" type="warning" :closable="false" :title="errorMessage" />
      <el-table v-loading="loading" :data="filteredRows" class="wide-table">
        <el-table-column prop="name" label="Agent" min-width="160" />
        <el-table-column prop="environmentName" label="绑定环境" min-width="160" />
        <el-table-column prop="version" label="版本" min-width="100" />
        <el-table-column prop="lastHeartbeatAt" label="心跳" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.lastHeartbeatAt) }}</template>
        </el-table-column>
        <el-table-column label="可执行能力" min-width="260">
          <template #default="{ row }">{{ joinCapabilities(row.capabilities) }}</template>
        </el-table-column>
        <el-table-column label="最近任务日志" min-width="180">
          <template #default="{ row }">{{ row.currentTaskId ?? 'heartbeat ok' }}</template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
      </el-table>
    </el-card>

    <AgentRegisterDrawer v-model:visible="drawerVisible" :environments="environments" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AgentRegisterDrawer from '@/components/AgentRegisterDrawer.vue'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { listAgents, type AgentInfo } from '@/api/agents'
import { listEnvironments, type EnvironmentInfo } from '@/api/environments'
import { formatDateTime, joinCapabilities } from '@/utils/format'

const keyword = ref('')
const drawerVisible = ref(false)
const agents = ref<AgentInfo[]>([])
const environments = ref<EnvironmentInfo[]>([])
const loading = ref(false)
const errorMessage = ref('')

const onlineCount = computed(() => agents.value.filter((item) => item.status === 'ONLINE').length)
const offlineCount = computed(() => agents.value.filter((item) => item.status === 'OFFLINE').length)
const runningCount = computed(() => agents.value.filter((item) => item.currentTaskId).length)

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
</style>
