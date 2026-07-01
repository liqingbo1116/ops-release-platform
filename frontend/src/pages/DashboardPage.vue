<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>首页工作台</h1>
        <p>以真实运行环境生成基线，驱动项目环境发布、镜像同步、健康检查与审计追踪。</p>
      </div>
      <div class="top-actions">
        <el-button :loading="loading" @click="loadData">刷新运行态</el-button>
        <el-button type="primary" @click="$router.push('/baselines')">生成环境基线</el-button>
      </div>
    </div>

    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon :closable="false" />

    <div class="metric-grid">
      <MetricCard label="在线环境" :value="healthyEnvCount" foot="连接正常" tone="good" />
      <MetricCard label="Agent 在线" :value="`${onlineAgentCount}/${agentStore.items.length}`" foot="心跳正常" tone="good" />
      <MetricCard label="已锁定基线" :value="lockedBaselineCount" foot="真实基线数据" />
      <MetricCard label="待处理部署" :value="pendingDeployCount" foot="部署任务状态" tone="warn" />
      <MetricCard label="执行中部署" :value="runningDeployCount" foot="部署任务状态" />
    </div>

    <div class="two-col">
      <el-card shadow="never">
        <template #header>
          <div class="panel-head">
            <strong>项目环境发布状态</strong>
            <el-button link type="primary" @click="$router.push('/compare')">查看差异</el-button>
          </div>
        </template>
        <el-table v-loading="loading" :data="environmentStore.items" class="wide-table">
          <el-table-column prop="name" label="产品" min-width="160" />
          <el-table-column label="Agent" min-width="120">
            <template #default="{ row }"><StatusTag :status="row.agentStatus" /></template>
          </el-table-column>
          <el-table-column prop="namespace" label="Namespace" min-width="140" show-overflow-tooltip />
          <el-table-column prop="registryProject" label="Harbor Project" min-width="140" show-overflow-tooltip />
          <el-table-column label="状态" min-width="100">
            <template #default="{ row }"><StatusTag :status="row.status" /></template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <div class="panel-head">
            <strong>部署与失败任务</strong>
            <el-button link type="primary" @click="$router.push('/deploy-tasks')">查看部署任务</el-button>
          </div>
        </template>
        <el-table v-loading="loading" :data="taskRows" class="wide-table">
          <el-table-column prop="id" label="任务" min-width="150" />
          <el-table-column prop="type" label="类型" min-width="90" />
          <el-table-column prop="currentStep" label="步骤" min-width="130" />
          <el-table-column label="结果" min-width="110">
            <template #default="{ row }"><StatusTag :status="row.status" /></template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listAgents } from '@/api/agents'
import { listBaselines } from '@/api/baselines'
import { listDeployTasks } from '@/api/deployTasks'
import { listEnvironments } from '@/api/environments'
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { useAgentStore } from '@/stores/agentStore'
import { useBaselineStore } from '@/stores/baselineStore'
import { useDeployStore } from '@/stores/deployStore'
import { useEnvironmentStore } from '@/stores/environmentStore'

const agentStore = useAgentStore()
const baselineStore = useBaselineStore()
const deployStore = useDeployStore()
const environmentStore = useEnvironmentStore()
const loading = ref(false)
const errorMessage = ref('')

const healthyEnvCount = computed(() => environmentStore.items.filter((item) => item.status === 'HEALTHY').length)
const onlineAgentCount = computed(() => agentStore.items.filter((item) => item.status === 'ONLINE').length)
const lockedBaselineCount = computed(() => baselineStore.items.filter((item) => item.status === 'LOCKED').length)
const pendingDeployCount = computed(
  () => deployStore.items.filter((item) => ['PENDING', 'WAITING', 'FAILED'].includes(item.status)).length,
)
const runningDeployCount = computed(() => deployStore.items.filter((item) => item.status === 'RUNNING').length)
const taskRows = computed(() => deployStore.items.filter((item) => item.status !== 'SUCCESS').slice(0, 8))

async function loadData() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [environments, agents, baselines, deployTasks] = await Promise.all([
      listEnvironments(),
      listAgents(),
      listBaselines(),
      listDeployTasks(),
    ])
    environmentStore.items = environments
    agentStore.items = agents
    baselineStore.items = baselines
    deployStore.items = deployTasks
  } catch (error) {
    environmentStore.items = []
    agentStore.items = []
    baselineStore.items = []
    deployStore.items = []
    errorMessage.value = error instanceof Error ? error.message : '首页运行态加载失败'
    ElMessage.error(errorMessage.value)
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>
