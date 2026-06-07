<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>首页工作台</h1>
        <p>以真实运行环境生成基线，驱动项目环境发布、镜像同步、健康检查与审计追踪。</p>
      </div>
      <div class="top-actions">
        <el-button>刷新运行态</el-button>
        <el-button type="primary" @click="$router.push('/baselines')">生成环境基线</el-button>
      </div>
    </div>

    <div class="metric-grid">
      <MetricCard label="在线环境" :value="healthyEnvCount" foot="连接正常" tone="good" />
      <MetricCard label="Agent 在线" :value="`${onlineAgentCount}/${mockData.agents.length}`" foot="心跳正常" tone="good" />
      <MetricCard label="已锁定基线" :value="lockedBaselineCount" foot="今日新增 1 个" />
      <MetricCard label="待发布服务" :value="mockData.diffResult.summary.publishable" foot="来自差异对比" tone="warn" />
      <MetricCard label="执行中发布" value="1" foot="健康检查阶段" />
    </div>

    <div class="two-col">
      <el-card shadow="never">
        <template #header>
          <div class="panel-head">
            <strong>项目环境发布状态</strong>
            <el-button link type="primary" @click="$router.push('/compare')">查看差异</el-button>
          </div>
        </template>
        <el-table :data="mockData.environments" class="wide-table">
          <el-table-column prop="name" label="环境" min-width="160" />
          <el-table-column label="Agent" min-width="120">
            <template #default="{ row }"><StatusTag :status="row.agentStatus" /></template>
          </el-table-column>
          <el-table-column label="服务数" min-width="100">
            <template #default>213</template>
          </el-table-column>
          <el-table-column label="基线一致率" min-width="120">
            <template #default="{ $index }">{{ $index === 0 ? '100%' : '68%' }}</template>
          </el-table-column>
          <el-table-column label="状态" min-width="100">
            <template #default="{ row }"><StatusTag :status="row.status" /></template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card shadow="never">
        <template #header>
          <div class="panel-head">
            <strong>部署与失败任务</strong>
            <el-button link type="primary" @click="$router.push('/deploy-tasks/DEP-20260607-009')">打开部署详情</el-button>
          </div>
        </template>
        <el-table :data="taskRows" class="wide-table">
          <el-table-column prop="id" label="任务" min-width="150" />
          <el-table-column prop="type" label="类型" min-width="90" />
          <el-table-column prop="step" label="步骤" min-width="130" />
          <el-table-column label="结果" min-width="110">
            <template #default="{ row }"><StatusTag :status="row.status" /></template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import MetricCard from '@/components/MetricCard.vue'
import StatusTag from '@/components/StatusTag.vue'
import { mockData } from '@/api/mockData'

const healthyEnvCount = mockData.environments.filter((item) => item.status === 'HEALTHY').length
const onlineAgentCount = mockData.agents.filter((item) => item.status === 'ONLINE').length
const lockedBaselineCount = mockData.baselines.filter((item) => item.status === 'LOCKED').length
const taskRows = [
  { id: 'REL-20260607-031', type: '发布', step: 'HTTP 健康检查', status: 'PARTIAL_FAILED' },
  { id: 'DEP-20260607-009', type: '部署', step: '恢复 MinIO', status: 'RUNNING' },
  { id: 'DEP-20260606-018', type: '部署', step: '健康检查', status: 'SUCCESS' },
]
</script>
