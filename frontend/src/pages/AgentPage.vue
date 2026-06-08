<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>Agent 管理</h1>
        <p>项目环境 Agent 主动拉取任务，执行镜像同步、kubectl、shell、健康检查并回传日志。</p>
      </div>
      <el-button type="primary" @click="drawerVisible = true">注册 Agent</el-button>
    </div>

    <div class="metric-grid six">
      <MetricCard label="注册 Agent" :value="agentMockData.agents.length" foot="绑定环境" />
      <MetricCard label="在线" :value="onlineCount" foot="心跳正常" tone="good" />
      <MetricCard label="执行中" :value="runningCount" foot="发布 / 部署" />
      <MetricCard label="离线" :value="offlineCount" foot="需排查" tone="bad" />
      <MetricCard label="平均心跳" value="18s" />
      <MetricCard label="版本覆盖" value="92%" />
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="搜索 Agent、环境、能力" clearable />
        <el-button>下发探测任务</el-button>
      </div>
      <el-table :data="filteredRows" class="wide-table">
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

    <AgentRegisterDrawer v-model:visible="drawerVisible" :environments="environmentMockData.environments" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import AgentRegisterDrawer from '@/components/AgentRegisterDrawer.vue'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { agentMockData } from '@/api/mockData/agent'
import { environmentMockData } from '@/api/mockData/environment'
import { formatDateTime, joinCapabilities } from '@/utils/format'

const keyword = ref('')
const drawerVisible = ref(false)
const onlineCount = agentMockData.agents.filter((item) => item.status === 'ONLINE').length
const offlineCount = agentMockData.agents.filter((item) => item.status === 'OFFLINE').length
const runningCount = agentMockData.agents.filter((item) => item.currentTaskId).length

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return agentMockData.agents
  return agentMockData.agents.filter((item) =>
    `${item.name} ${item.environmentName} ${item.capabilities.join(' ')}`.toLowerCase().includes(q),
  )
})
</script>
